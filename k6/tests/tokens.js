import { check, fail, sleep } from 'k6';
import exec from 'k6/execution';
import http from 'k6/http';
import { getCookies, login, generateAuthorizationHeader, addMinutes } from "../rancher/rancher_utils.js";
import { getPrincipalIds, getCurrentUserId, getClusterIds, getCurrentUserPrincipal, createUser } from "../rancher/rancher_users_utils.js"


// Parameters
const tokenCount = Number(__ENV.TOKEN_COUNT)
const vus = Number(__ENV.K6_VUS) || 1
const perVuIterations = __ENV.PER_VU_ITERATIONS


// Option setting
const baseUrl = __ENV.BASE_URL
const username = __ENV.USERNAME
const password = __ENV.PASSWORD

// Option setting
export const options = {
  insecureSkipTLSVerify: true,

  setupTimeout: '8h',

  scenarios: {
    useTokens: {
      executor: 'shared-iterations',
      exec: 'useTokens',
      vus: vus,
      iterations: perVuIterations,
      maxDuration: '1h',
    },
  },
  // thresholds: {
  //   checks: ['rate>0.99']
  // }
  thresholds: {
    http_req_failed: ['rate<=0.01'], // http errors should be less than 1%
    http_req_duration: ['p(99)<=500'], // 95% of requests should be below 500ms
    checks: ['rate>0.99'], // the rate of successful checks should be higher than 99%
  }
}

export function setup() {
  // log in
  if (!login(baseUrl, {}, username, password)) {
    fail(`could not login into cluster`)
  }
  const cookies = getCookies(baseUrl)

  // delete leftovers, if any
  cleanup(cookies)

  // return data that remains constant throughout the test
  let data = {
    cookies: cookies,
    principalIds: getPrincipalIds(baseUrl, cookies),
    myUserId: getCurrentUserId(baseUrl, cookies),
    myUserPrincipal: getCurrentUserPrincipal(baseUrl, cookies),
    clusterIds: getClusterIds(baseUrl, cookies),
    fullTokens: []
  }
  let createdTokens = []
  for (let i = 0; i < tokenCount; i++) {
    createdTokens.push(createToken(baseUrl, data, i))
  }

  createdTokens.forEach(r => {
    data["fullTokens"].push(r)
  })
  return data
}

export function daysToMilliseconds(days) {
  return days * 24 * 60 * 60 * 1000;
}

function createToken(baseUrl, data, idx) {
  const clusterId = data.clusterIds[idx % data.clusterIds.length]
  const id = `dartboard-${idx}`
  const ttl = daysToMilliseconds(90) // Default ttl from UI is 90 days, but the API uses milliseconds

  const body = {
    // "clusterId": clusterId,
    // "current": false,
    "description": `Dartboard test token ${id}`,
    // "enabled": true,
    // "expired": false,
    // "isDerived": true,
    // "userId": data.myUserId,
    // "userPrincipal": data.myUserPrincipal.id,
    "metadata": {},
    "ttl": ttl,
    "type": "token",
  }

  const res = http.post(
    `${baseUrl}/v3/tokens`,
    JSON.stringify(body),
    { cookies: data.cookies }
  )

  let token = JSON.parse(res.body)

  check(res, {
    'POST v3/tokens returns status 201': (r) => r.status === 201,
  })

  return JSON.parse(res.body)
}

export function deleteTokens(data) {
  let tokensData = getTokens(baseUrl, data)
  tokensData.filter(r => ("description" in r) && r["description"].startsWith("Dartboard test")).forEach(r => {
    let res = http.del(`${baseUrl}/v3/tokens/${r["id"]}`, { cookies: data.cookies })
    check(res, {
      'DELETE /v3/tokens returns status 200': (r) => r.status === 200 || r.status === 204,
    })
  })
}

export function getTokens(data) {
  let res = http.get(`${baseUrl}/v3/tokens`, { cookies: data.cookies })
  check(res, {
    'GET /v3/tokens returns status 200': (r) => r.status === 200 || r.status === 204,
  })
  return JSON.parse(res.body)["data"]
}

export function getTokensByClusterID(data, clusterId) {
  const clusterIdParam = `?clusterId=${clusterId}`
  const checkString = `GET /v3/tokens${clusterIdParam} returns status 200`
  let res = http.get(`${baseUrl}/v3/tokens${clusterIdParam}`, { cookies: data.cookies })
  check(res, {
    checkString: (r) => r.status === 200 || r.status === 204,
  })
}

function getUsers(baseUrl, params) {
  let res = http.get(`${baseUrl}/v3/users`, params);
  check(res, { 'GET users returns status 200': (r) => r.status === 200, });
  return res;
}

function useToken(baseUrl, bearerToken) {
  let params = {
    insecureSkipTLSVerify: true,
    ...generateAuthorizationHeader(bearerToken)
  };
  let res = getUsers(baseUrl, params)
  check(res, { 'auth with bearer token': (r) => r.status === 200, });
  return res, params
}

export function useTokens(data) {
  let tokensData = getTokens(data);
  let tokens = tokensData.filter(r => ("description" in r) && r["description"].startsWith("Dartboard test"));
  let lastUsedAtTS = {};
  // Use the tokens for the first time
  tokens.forEach(token => {
    let bearerToken = data["fullTokens"].find(f => token.id == f.id).token;
    useToken(baseUrl, bearerToken)
  });
  tokensData = getTokens(data);
  tokens = tokensData.filter(r => ("description" in r) && r["description"].startsWith("Dartboard test"));

  // Track lastUsedAtTS for each token
  tokens.forEach(token => {
    // Make sure we're storing a valid number
    const timestamp = Number(token.lastUsedAtTS);
    if (isNaN(timestamp)) {
      console.error(`Invalid timestamp for token ${token.id}:`, token.lastUsedAtTS);
    };
    lastUsedAtTS[token.id] = timestamp;
  });

  // Utilize each token and verify that lastUsedAtTS is updated to a newer timestamp
  tokens.forEach(token => {
    let bearerToken = data["fullTokens"].find(f => token.id == f.id).token;
    let _, params = useToken(baseUrl, bearerToken)
    createUser(baseUrl, params, `${token.id}-${exec.scenario.iterationInTest}`);
  });

  tokensData = getTokens(data);
  tokens = tokensData.filter(r => ("description" in r) && r["description"].startsWith("Dartboard test"));
  check(true, {
    'all tokens lastUsedAtTS are newer after utilizing them': () =>
      tokens.every(r => r.lastUsedAtTS > lastUsedAtTS[r.id])
  });
}

function cleanup(data) {
  deleteTokens(data);
  let res = getUsers(baseUrl, { cookies: data.cookies });
  let users = JSON.parse(res.body)["data"];
  users.filter(r => r["description"].startsWith("Dartboard Test User ")).forEach(r => {
    let res = http.del(`${baseUrl}/v3/users/${r["id"]}`, { cookies: data.cookies });
    check(res, {
      'DELETE /v3/users returns status 200': (r) => r.status === 200 || r.status === 204,
    });
  })
}
