import { check, fail, sleep } from 'k6';
import http from 'k6/http'
import { retryUntilExpected } from "../rancher/rancher_utils.js";
import * as YAML from '../lib/js-yaml-4.1.0.mjs'

export const baseNamespacesPath = "v1/namespaces"
export const namespaceTag = { url: `/v1/namespaces/<Namespace ID>` }
export const namespacesTag = { url: `/v1/namespaces` }
export const postNamespaceTag = { url: `/v1/namespaces` }
export const putNamespaceTag = { url: `/v1/namespaces/<Namespace ID>` }


export function cleanupMatchingNamespaces(baseUrl, cookies, namePrefix) {
  let res = http.get(`${baseUrl}/${baseNamespacesPath}`, { cookies: cookies })
  check(res, {
    '/v1/management.cattle.io.namespaces returns status 200': (r) => r.status === 200,
  })
  JSON.parse(res.body)["data"].filter(r => r["metadata"]["name"].startsWith(namePrefix)).forEach(r => {
    res = http.del(`${baseUrl}/${baseNamespacesPath}/${r["id"]}`, { cookies: cookies })
    check(res, {
      'DELETE /v3/namespaces returns status 200': (r) => r.status === 200,
    })
  })
}

export function getNamespace(baseUrl, cookies, id) {
  let res = http.get(`${baseUrl}/${baseNamespacesPath}`, { cookies: cookies, tag: namespacesTag })
  let criteria = []
  criteria[`GET /${baseNamespacesPath} returns status 200`] = (r) => r.status === 200
  check(res, criteria)
  namespaceArray = JSON.parse(res.body)["data"].filter(r => r["id"] == id)
  return { res: res, namespace: namespaceArray[0] }
}

export function getNamespaces(baseUrl, cookies) {
  let res = http.get(`${baseUrl}/${baseNamespacesPath}`, { cookies: cookies, tags: namespacesTag })
  check(res, {
    'GET /v1/management.cattle.io.namespaces returns status 200': (r) => r.status === 200,
  })
  let namespaceArray = JSON.parse(res.body)["data"]
  return { res: res, namespaceArray: namespaceArray }
}

export function createNamespace(baseUrl, body, cookies) {
  const res = http.post(
    `${baseUrl}/${baseNamespacesPath}`,
    body,
    {
      headers: {
        "content-type": "application/json",
      },
      cookies: cookies,
      tags: postNamespaceTag,
    }
  )
  check(res, {
    'Namespace post returns 201 (created)': (r) => r.status === 201,
  })
  return res
}

export function deleteNamespace(baseUrl, cookies, id) {
  let res = http.del(`${baseUrl}/${baseNamespacesPath}/${id}`, null, { cookies: cookies, tags: namespaceTag })

  check(res, {
    'DELETE /v3/namespaces returns status 200': (r) => r.status === 200 || r.status === 204,
  })
  return res
}

export function getNamespacesMatchingName(baseUrl, cookies, namePrefix) {
  let { res, namespaceArray } = getNamespaces(baseUrl, cookies)
  if (!namespaceArray || namespaceArray.length == 0) {
    console.log("Could not get list of Namespaces");
    return { res: res, namespaceArray: [] };
  }
  let filteredArray = []
  filteredArray = namespaceArray.filter(r => r["metadata"]["name"].startsWith(namePrefix))
  return { res: res, namespaceArray: filteredArray }
}

export function updateNamespace(baseUrl, cookies, namespace) {
  let body = YAML.dump(namespace)
  let res = http.put(
    `${baseUrl}/${baseNamespacesPath}/${namespace.metadata.name}`,
    body,
    {
      headers: {
        accept: 'application/json',
        'content-type': 'application/yaml',
      },
      cookies: cookies,
      tags: putNamespaceTag,
    }
  )
  check(res, {
    'PUT /v3/namespaces/<Namespace ID> returns status 200': (r) => r.status === 200,
  })
  return res
}
