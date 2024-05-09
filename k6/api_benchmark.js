import { check, fail } from 'k6';
import http from 'k6/http';

// Parameters
const vus = __ENV.VUS || 1
const perVuIterations = __ENV.PER_VU_ITERATIONS || 30
const baseUrl = __ENV.BASE_URL
const username = __ENV.USERNAME
const password = __ENV.PASSWORD
const token = __ENV.TOKEN
const cluster = __ENV.CLUSTER || "local"
const resource = __ENV.RESOURCE || "management.cattle.io.setting"
const api = __ENV.API || "steve"
const paginationStyle = __ENV.PAGINATION_STYLE || "k8s"
const pageSize = __ENV.PAGE_SIZE || 100
const urlSuffix = __ENV.URL_SUFFIX || ""

// Option setting
export const options = {
    insecureSkipTLSVerify: true,

    scenarios: {
        list : {
            executor: 'per-vu-iterations',
            exec: 'list',
            vus: vus,
            iterations: perVuIterations,
            maxDuration: '24h',
        }
    },
    thresholds: {
        checks: ['rate>0.99']
    }
}

// Test functions, in order of execution

export function setup() {
    // if session cookie was specified, save it
    if (token) {
        return {R_SESS: token}
    }

    // if credentials were specified, log in
    if (username && password) {
        const res = http.post(`${baseUrl}/v3-public/localProviders/local?action=login`, JSON.stringify({
            "description": "UI session",
            "responseType": "cookie",
            "username": username,
            "password": password
        }))

        check(res, {
            'logging in returns status 200': (r) => r.status === 200,
        })

        return http.cookieJar().cookiesForURL(res.url)
    }

    return {}
}



export function list(cookies) {
    if (api === "steve") {
        const url = cluster === "local"?
            `${baseUrl}/v1/${resource}` :
            `${baseUrl}/k8s/clusters/${cluster}/v1/${resource}`

        if (paginationStyle === "k8s") {
            listWithK8sStylePagination(url, cookies)
        }
        else if (paginationStyle === "steve") {
            listWithSteveStylePagination(url, cookies)
        }
        else {
            fail("Invalid PAGINATION_STYLE value: " + paginationStyle)
        }
    }
    else if (api === "norman") {
        const url = `${baseUrl}/v3/${resource}`
        listWithNormanStylePagination(url, cookies)
    }
    else {
        fail("Invalid API value: " + api)
    }
}

function listWithK8sStylePagination(url, cookies) {
    let revision = null
    let continueToken = null
    while (true) {
        const fullUrl = url + "?limit=" + pageSize +
            (revision != null ? "&revision=" + revision : "") +
            (continueToken != null ? "&continue=" + continueToken : "") +
            urlSuffix

        const res = http.get(fullUrl, {cookies: cookies})

        const criteria = {}
        criteria[`listing ${resource} from cluster ${cluster} (steve with k8s style pagination) succeeds`] = (r) => r.status === 200
        criteria[`no slow pagination errors (410 Gone) detected`] = (r) => r.status !== 410
        check(res, criteria)

        try {
            const body = JSON.parse(res.body)
            if (body === undefined || body.continue === undefined) {
                break
            }
            if (revision == null) {
                revision = body.revision
            }
            continueToken = body.continue
        } catch (e) {
            if (e instanceof SyntaxError) {
                fail("Response body does not parse as JSON: " + res.body)
            }
            throw e
        }
    }
}

function listWithSteveStylePagination(url, cookies) {
    let i = 1
    let revision = null
    while (true) {
        const fullUrl = url + "?pagesize=" + pageSize + "&page=" + i +
            (revision != null ? "&revision=" + revision : "") +
            urlSuffix

        const res = http.get(fullUrl, {cookies: cookies})

        const criteria = {}
        criteria[`listing ${resource} from cluster ${cluster} (steve style pagination) succeeds`] = (r) => r.status === 200
        criteria[`no slow pagination errors (410 Gone) detected`] = (r) => r.status !== 410
        check(res, criteria)

        try {
            const body = JSON.parse(res.body)
            if (body === undefined || body.data === undefined || body.data.length === 0) {
                break
            }
            if (revision == null) {
                revision = body.revision
            }
            i = i + 1
        }
        catch (e){
            if (e instanceof SyntaxError) {
                fail("Response body does not parse as JSON: " + res.body)
            }
            throw e
        }
    }
}

function listWithNormanStylePagination(url, cookies) {
    let nextUrl = url + "?limit=" + pageSize
    while (true) {
        const res = http.get(nextUrl, {cookies: cookies})

        const criteria = {}
        criteria[`listing ${resource} from cluster ${cluster} (norman style pagination) succeeds`] = (r) => r.status === 200
        criteria[`no slow pagination errors (410 Gone) detected`] = (r) => r.status !== 410
        check(res, criteria)

        try {
            const body = JSON.parse(res.body)
            if (body === undefined || body.pagination === undefined || body.pagination.partial === undefined || body.pagination.next === undefined) {
                break
            }
            nextUrl = body.pagination.next
        } catch (e) {
            if (e instanceof SyntaxError) {
                fail("Response body does not parse as JSON: " + res.body)
            }
            throw e
        }
    }
}
