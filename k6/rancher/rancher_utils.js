import { check, fail, sleep } from 'k6'
import http from 'k6/http'
import { getCurrentUserId, getUserPreferences, setUserPreferences } from "./rancher_users_utils.js"
import encoding from 'k6/encoding';


export function getCookies(baseUrl) {
  const response = http.get(`${baseUrl}/`)
  return http.cookieJar().cookiesForURL(response.url)
}

export function login(baseUrl, cookies, username, password) {
  const params = {
    headers: {
      accept: 'application/json',
      'content-type': 'application/json; charset=UTF-8',
    },
  }
  
  // Only add cookies if they exist and are not empty
  if (cookies && Object.keys(cookies).length > 0) {
    console.log("Using cookies: ", cookies)
    params.cookies = cookies
  }
  const response = http.post(
    `${baseUrl}/v3-public/localProviders/local?action=login`,
    JSON.stringify({ "description": "UI session", "responseType": "cookie", "username": username, "password": password }),
    params
  )

  check(response, {
    'login works': (r) => r.status === 200 || r.status === 401,
  })

  return response
}

export function firstLogin(baseUrl, cookies, bootstrapPassword, password) {
  let response

  if (login(baseUrl, cookies, "admin", bootstrapPassword).status === 200) {
    response = http.post(
      `${baseUrl}/v3/users?action=changepassword`,
      JSON.stringify({ "currentPassword": bootstrapPassword, "newPassword": password }),
      {
        headers: {
          accept: 'application/json',
          'content-type': 'application/json; charset=UTF-8',
        },
        cookies: { cookies },
      }
    )
    check(response, {
      'password can be changed': (r) => r.status === 200,
    })
    if (response.status !== 200) {
      fail('first password change was not successful')
    }
  }
  else {
    console.warn("bootstrap password already changed")
    if (login(baseUrl, cookies, "admin", password).status !== 200) {
      fail('neither bootstrap nor normal passwords were accepted')
    }
  }
  const userId = getCurrentUserId(baseUrl, cookies)
  const userPreferences = getUserPreferences(baseUrl, cookies);

  userPreferences["data"]["locale"] = "en-us"
  setUserPreferences(baseUrl, cookies, userId, userPreferences);

  response = http.get(
    `${baseUrl}/v1/management.cattle.io.settings`,
    {
      headers: {
        accept: 'application/json',
      },
      cookies: { cookies },
    }
  )
  check(response, {
    'Settings can be queried': (r) => r.status === 200,
  })
  const settings = JSON.parse(response.body)

  const firstLoginSetting = settings.data.filter(d => d.id === "first-login")[0]
  if (firstLoginSetting === undefined) {
    response = http.post(
      `${baseUrl}/v1/management.cattle.io.settings`,
      JSON.stringify({ "type": "management.cattle.io.setting", "metadata": { "name": "first-login" }, "value": "false" }),
      {
        headers: {
          accept: 'application/json',
          'content-type': 'application/json',
        },
        cookies: { cookies },
      }
    )
    check(response, {
      'First login setting can be set': (r) => r.status === 201,
    })
  }
  else {
    firstLoginSetting["value"] = "false"
    response = http.put(
      `${baseUrl}/v1/management.cattle.io.settings/first-login`,
      JSON.stringify(firstLoginSetting),
      {
        headers: {
          accept: 'application/json',
          'content-type': 'application/json',
        },
        cookies: { cookies },
      }
    )
    check(response, {
      'First login setting can be changed': (r) => r.status === 200,
    })
  }

  const eulaSetting = settings.data.filter(d => d.id === "eula-agreed")[0]
  if (eulaSetting === undefined) {
    response = http.post(
      `${baseUrl}/v1/management.cattle.io.settings`,
      JSON.stringify({ "type": "management.cattle.io.setting", "metadata": { "name": "eula-agreed" }, "value": timestamp(), "default": timestamp() }),
      {
        headers: {
          accept: 'application/json',
          'content-type': 'application/json',
        },
        cookies: { cookies },
      }
    )
    check(response, {
      'EULA setting can be set': (r) => r.status === 201,
    })
  }
  else {
    eulaSetting["value"] = timestamp()
    response = http.put(
      `${baseUrl}/v1/management.cattle.io.settings/eula-agreed`,
      JSON.stringify(eulaSetting),
      {
        headers: {
          accept: 'application/json',
          'content-type': 'application/json',
        },
        cookies: { cookies },
      }
    )
    check(response, {
      'EULA setting can be changed': (r) => r.status === 200,
    })
  }

  const telemetrySetting = settings.data.find(d => d.id === "telemetry-opt")
  if (telemetrySetting === undefined) {
    response = http.post(
      `${baseUrl}/v1/management.cattle.io.settings/telemetry-opt`,
      JSON.stringify({ "type": "management.cattle.io.setting", "metadata": { "name": "telemetry-opt", "value": "out" } }),
      {
        headers: {
          accept: 'application/json',
          'content-type': 'application/json',
        },
        cookies: { cookies },
      }
    )
    check(response, {
      'telemetry setting can be set': (r) => r.status === 201,
    })
  }
  else {
    telemetrySetting["value"] = "out"
    response = http.put(
      `${baseUrl}/v1/management.cattle.io.settings/telemetry-opt`,
      JSON.stringify(telemetrySetting),
      {
        headers: {
          accept: 'application/json',
          'content-type': 'application/json',
        },
        cookies: { cookies },
      }
    )
    check(response, {
      'telemetry setting can be changed': (r) => r.status === 200,
    })
  }
}

export function timestamp() {
  return new Date().toISOString()
}

export function addMinutes(date, minutes) {
  return new Date(date.getTime() + (minutes*60000));
}

export function createImportedCluster(baseUrl, cookies, name) {
  let response

  const userId = getCurrentUserId(baseUrl, cookies)
  const userPreferences = getUserPreferences(baseUrl, cookies);

  userPreferences["last-visited"] = "{\"name\":\"c-cluster-product\",\"params\":{\"cluster\":\"_\",\"product\":\"manager\"}}"
  userPreferences["locale"] = "en-us"
  userPreferences["seen-whatsnew"] = "\"v2.7.1\""
  userPreferences["seen-cluster"] = "_"
  setUserPreferences(baseUrl, cookies, userId, userPreferences)

  response = http.get(`${baseUrl}/v1/catalog.cattle.io.clusterrepos`, {
    headers: {
      accept: 'application/json',
    },
    cookies: cookies,
  })
  check(response, {
    'querying clusterrepos works': (r) => r.status === 200,
  })

  response = http.get(`${baseUrl}/v1/management.cattle.io.kontainerdrivers`, {
    headers: {
      accept: 'application/json',
      cookies: cookies,
    },
  })
  check(response, {
    'querying kontainerdrivers works': (r) => r.status === 200,
  })

  response = http.get(
    `${baseUrl}/v1/catalog.cattle.io.clusterrepos/rancher-charts?link=index`,
    {
      headers: {
        accept: 'application/json',
      },
      cookies: cookies,
    }
  )
  check(response, {
    'querying rancher-charts works': (r) => r.status === 200,
  })

  response = http.get(
    `${baseUrl}/v1/catalog.cattle.io.clusterrepos/rancher-partner-charts?link=index`,
    {
      headers: {
        accept: 'application/json',
      },
      cookies: cookies,
    }
  )
  check(response, {
    'querying rancher-partners-charts works': (r) => r.status === 200,
  })

  response = http.get(
    `${baseUrl}/v1/catalog.cattle.io.clusterrepos/rancher-rke2-charts?link=index`,
    {
      headers: {
        accept: 'application/json',
      },
      cookies: cookies,
    }
  )
  check(response, {
    'querying rancher-rke2-charts works': (r) => r.status === 200,
  })

  response = http.get(`${baseUrl}/v3/clusterroletemplatebindings`, {
    headers: {
      accept: 'application/json',
    },
    cookies: cookies,
  })
  check(response, {
    'querying clusterroletemplatebindings works': (r) => r.status === 200,
  })

  response = http.get(`${baseUrl}/v1/management.cattle.io.roletemplates`, {
    headers: {
      accept: 'application/json',
    },
    cookies: cookies,
  })
  check(response, {
    'querying roletemplates works': (r) => r.status === 200,
  })

  response = http.post(
    `${baseUrl}/v1/provisioning.cattle.io.clusters`,
    JSON.stringify({ "type": "provisioning.cattle.io.cluster", "metadata": { "namespace": "fleet-default", "name": name }, "spec": {} }),
    {
      headers: {
        accept: 'application/json',
        'content-type': 'application/json',
      },
      cookies: cookies,
    }
  )

  check(response, {
    'creating an imported cluster works': (r) => r.status === 201 || r.status === 409,
  })
  if (response.status === 409) {
    console.warn(`cluster ${name} already exists`)
  }

  response = http.get(
    `${baseUrl}/v1/provisioning.cattle.io.clusters/fleet-default/${name}`,
    {
      headers: {
        accept: 'application/json',
      },
      cookies: cookies,
    }
  )
  check(response, {
    'querying clusters works': (r) => r.status === 200,
  })
  if (!response.status === 200) {
    fail(`cluster ${name} not found`)
  }

  response = http.get(`${baseUrl}/v1/cluster.x-k8s.io.machinedeployments`, {
    headers: {
      accept: 'application/json',
    },
    cookies: cookies,
  })
  check(response, {
    'querying machinedeployments works': (r) => r.status === 200,
  })

  response = http.get(`${baseUrl}/v1/rke.cattle.io.etcdsnapshots`, {
    headers: {
      accept: 'application/json',
    },
    cookies: cookies,
  })
  check(response, {
    'querying etcdsnapshots works': (r) => r.status === 200,
  })

  response = http.get(`${baseUrl}/v1/management.cattle.io.nodetemplates`, {
    headers: {
      accept: 'application/json',
    },
    cookies: cookies,
  })
  check(response, {
    'querying nodetemplates works': (r) => r.status === 200,
  })

  response = http.get(`${baseUrl}/v1/management.cattle.io.clustertemplates`, {
    headers: {
      accept: 'application/json',
    },
    cookies: cookies,
  })
  check(response, {
    'querying clustertemplates works': (r) => r.status === 200,
  })

  response = http.get(
    `${baseUrl}/v1/management.cattle.io.clustertemplaterevisions`,
    {
      headers: {
        accept: 'application/json',
      },
      cookies: cookies,
    }
  )
  check(response, {
    'querying clustertemplaterevisions works': (r) => r.status === 200,
  })
}

export function logout(baseUrl, cookies) {
  const response = http.post(`${baseUrl}/v3/tokens?action=logout`, '{}', {
    headers: {
      accept: 'application/json',
      'content-type': 'application/json',
    },
    cookies: cookies
  })

  check(response, {
    'logging out works': (r) => r.status === 200,
  })

  return response
}

export function generateAuthorizationHeader(token) {
  return {
    headers: {
      'Authorization': `Basic ${encoding.b64encode(token)}`,
    }
  }
}

// Retries result-returning function for up to 10 times
// until a non-409 status is returned, waiting for up to 1s
// between retries
export function retryOnConflict(attempts=9, f) {
  for (let i = 0; i < 9; i++) {
    const res = f()
    if (res.status !== 409) {
      return res
    }
    console.warn(`attempt #${attempts + 1} failed with status ${res.status}`)
    // expected conflict. Sleep a bit and retry
    sleep(Math.random())
  }
  // all previous attempts failed, try one last time
  return f()
}

export function retryUntilExpected(expectedStatus, attempts=9, f) {
  for (let i = 0; i < attempts; i++) {
    const res = f()
    if (res.status === expectedStatus) {
      return res
    }
    console.warn(`attempt #${attempts + 1} failed with status ${res.status}`)
    // status doesn't match expected. Sleep a bit and retry
    sleep(Math.random())
  }
  // all previous attempts failed, try one last time
  return f()
}

export function retryUntilOneOf(expectedStatuses=[200, 201, 204], attempts=9, f) {
  for (let i = 0; i < attempts; i++) {
    const res = f()
    if (expectedStatuses.includes(res.status)) {
      return res
    }
    console.warn(`attempt #${attempts + 1} failed with status ${res.status}`)
    // status doesn't match expected. Sleep a bit and retry
    sleep(Math.random())
  }
  // all previous attempts failed, try one last time
  return f()
}
