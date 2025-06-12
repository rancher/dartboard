import { check, fail, sleep } from 'k6'
import http from 'k6/http'

export function getUserId(baseUrl, cookies) {
  let response = http.get(`${baseUrl}/v3/users?me=true`, {
      headers: {
          accept: 'application/json',
      },
      cookies: {cookies},
  })
  check(response, {
      'reading user details was successful': (r) => r.status === 200,
  })
  if (response.status !== 200) {
      fail('could not query user details')
  }

  return JSON.parse(response.body).data[0].id
}

export function getUserPreferences(baseUrl, cookies) {
  let response = http.get(`${baseUrl}/v1/userpreferences`, {
      headers: {
          accept: 'application/json',
      },
      cookies: {cookies},
  })
  check(response, {
      'preferences can be queried': (r) => r.status === 200,
  })
  return JSON.parse(response.body)["data"][0]
}

export function setUserPreferences(baseUrl, cookies, userId, userPreferences) {
  let response = http.put(
      `${baseUrl}/v1/userpreferences/${userId}`,
      JSON.stringify(userPreferences),
      {
          headers: {
              accept: 'application/json',
              'content-type': 'application/json',
          },
          cookies: {cookies},
      }
  )
  check(response, {
      'preferences can be set': (r) => r.status === 200,
  })
  return response;
}

export function getPrincipalIds(baseUrl, cookies) {
  const response = http.get(
      `${baseUrl}/v1/management.cattle.io.users`,
      {cookies: cookies}
  )
  if (response.status !== 200) {
      fail('could not list users')
  }
  const users = JSON.parse(response.body).data
  return users.filter(u => u["username"] != null).map(u => u["principalIds"][0])
}

export function getPrincipalById(baseUrl, cookies, id) {
  const response = http.get(
      `${baseUrl}/v3/principals/${id}`,
      {cookies: cookies}
  )
  if (response.status !== 200) {
      fail('could not get principal by ID')
  }
  return JSON.parse(response.body).data[0]
}

export function getUsers(baseUrl, params = null) {
  const response = http.get(`${baseUrl}/v3/users`, params);
  console.log("GET users status: ", response.status);
  check(response, {'status is 200': (r) => r.status === 200,}) || fail('could not get list of users');
  return JSON.parse(response.body)["data"];
}

export function getCurrentUserPrincipal(baseUrl, cookies) {
  const response = http.get(
      `${baseUrl}/v3/principals?me=true`,
      {cookies: cookies}
  )
  if (response.status !== 200) {
      fail('could not get current User\'s Principal')
  }
  return JSON.parse(response.body).data[0]
}

export function getCurrentUserId(baseUrl, cookies) {
  const response = http.get(
      `${baseUrl}/v3/users?me=true`,
      {cookies: cookies}
  )
  if (response.status !== 200) {
      fail('could not get my user')
  }
  return JSON.parse(response.body).data[0].id
}

export function getCurrentUserPrincipalId(baseUrl, cookies) {
  const response = http.get(
      `${baseUrl}/v3/users?me=true`,
      {cookies: cookies}
  )
  if (response.status !== 200) {
      fail('could not get my user')
  }
  return JSON.parse(response.body).data[0].principalIds[0]
}

export function getClusterIds(baseUrl, cookies) {
  const response = http.get(
      `${baseUrl}/v1/management.cattle.io.clusters`,
      {cookies: cookies}
  )
  if (response.status !== 200) {
      fail('could not list clusters')
  }
  const clusters = JSON.parse(response.body).data
  return clusters.map(c => c["id"])
}

export function createUser(baseUrl, params = null, id) {
    const res = http.post(`${baseUrl}/v3/users`,
        JSON.stringify({
            "type": "user",
            "name": `Dartboard Test User ${id}`,
            "description": `Dartboard Test User ${id}`,
            "enabled": true,
            "mustChangePassword": false,
            "password": "useruseruser",
            "username": `user-${id}`
        }),
        params
    )

    sleep(0.1)
    if (res.status != 201) {
        console.log("CREATE user failed:\n", JSON.stringify(res, null, 2))
    }
    check(res, {
        '/v3/users returns status 201': (r) => r.status === 201,
    })
    return JSON.parse(res.body)
}
