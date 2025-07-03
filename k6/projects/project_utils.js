import { check, fail, sleep } from 'k6';
import http from 'k6/http'
import { retryUntilExpected } from "../rancher/rancher_utils.js";
import * as YAML from '../lib/js-yaml-4.1.0.mjs'

export const baseProjectsPath = "v1/management.cattle.io.projects"
export const normanProjectsPath = "v3/projects"
export const normanProjectTag = { url: `/v3/projects/<Project ID>` }
export const normanProjectsTag = { url: `/v3/projects` }
export const projectsTag = { url: `/v1/management.cattle.io.projects` }
export const postProjectTag = { url: `/v3/projects` }
export const putProjectTag = { url: `/v3/projects/<Project ID>` }


export function cleanupMatchingProjects(baseUrl, cookies, namePrefix) {
  let res = http.get(`${baseUrl}/${baseProjectsPath}`, { cookies: cookies })
  check(res, {
    '/v1/management.cattle.io.projects returns status 200': (r) => r.status === 200,
  })
  JSON.parse(res.body)["data"].filter(r => r["spec"]["displayName"].startsWith(namePrefix)).forEach(r => {
    res = http.del(`${baseUrl}/${normanProjectsPath}/${r["id"]}`, { cookies: cookies })
    check(res, {
      'DELETE /v3/projects returns status 200': (r) => r.status === 200,
    })
  })
}

export function getProject(baseUrl, cookies, id) {
  let res = http.get(`${baseUrl}/${baseProjectsPath}`, { cookies: cookies, tag: projectsTag })
  let criteria = []
  criteria[`GET /${baseProjectsPath} returns status 200`] = (r) => r.status === 200
  check(res, criteria)
  let projectArray = JSON.parse(res.body)["data"].filter(r => r["id"] == id)
  return { res: res, project: projectArray[0] }
}

export function getProjects(baseUrl, cookies) {
  let res = http.get(`${baseUrl}/${baseProjectsPath}`, { cookies: cookies, tags: projectsTag })
  check(res, {
    'GET /v1/management.cattle.io.projects returns status 200': (r) => r.status === 200,
  })
  let projectArray = JSON.parse(res.body)["data"]
  return { res: res, projectArray: projectArray }
}

export function getNormanProjects(baseUrl, cookies) {
  let res = http.get(`${baseUrl}/${normanProjectsPath}`, { cookies: cookies, tags: normanProjectsTag })
  check(res, {
    'GET /v3/projects returns status 200': (r) => r.status === 200,
  })
  let projectArray = JSON.parse(res.body)["data"]
  return { res: res, projectArray: projectArray }
}

export function createNormanProject(baseUrl, body, cookies) {
  const res = http.post(
    `${baseUrl}/${normanProjectsPath}`,
    body,
    {
      headers: {
        "content-type": "application/json",
      },
      cookies: cookies,
      tags: postProjectTag,
    }
  )
  check(res, {
    'Project post returns 201 (created)': (r) => r.status === 201,
  })
  return res
}

export function deleteNormanProject(baseUrl, cookies, id) {
  let res = http.del(`${baseUrl}/${normanProjectsPath}/${id}`, null, { cookies: cookies, tags: normanProjectTag })

  check(res, {
    'DELETE /v3/projects returns status 200': (r) => r.status === 200 || r.status === 204,
  })
  return res
}

export function getProjectsMatchingName(baseUrl, cookies, namePrefix) {
  let { res, projectArray } = getProjects(baseUrl, cookies)
  if (!projectArray || projectArray.length === 0) {
    console.log("Could not get list of Projects");
    return { res: res, projectArray: [] };
  }
  let filteredArray = projectArray.filter(r => r["spec"]["displayName"].startsWith(namePrefix))
  return { res: res, projectArray: filteredArray }
}

export function getNormanProjectsMatchingName(baseUrl, cookies, namePrefix) {
  let { res, projectArray } = getNormanProjects(baseUrl, cookies)
  if (!projectArray || projectArray.length === 0) {
    console.log("Could not get list of Projects");
    return { res: res, projectArray: [] };
  }
  let filteredArray = projectArray.filter(r => r["name"].startsWith(namePrefix))
  return { res: res, projectArray: filteredArray }
}

export function updateProject(baseUrl, cookies, project) {
  let body = YAML.dump(project)
  let res = http.put(
    `${baseUrl}/${normanProjectsPath}/${project.metadata.name}`,
    body,
    {
      headers: {
        accept: 'application/json',
        'content-type': 'application/yaml',
      },
      cookies: cookies,
      tags: putProjectTag,
    }
  )
  check(res, {
    'PUT /v3/projects/<Project ID> returns status 200': (r) => r.status === 200,
  })
  return res
}
