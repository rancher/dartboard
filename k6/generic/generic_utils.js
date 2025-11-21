import {check, sleep} from 'k6';
import http from 'k6/http';

// Required params: baseurl, cookies, data, clusterId, namespace, iter
export function createConfigMaps(baseUrl, cookies, data, clusterId, namespace, iter) {
    const name = `test-config-map-${iter}`

    const url = clusterId === "local"?
      `${baseUrl}/v1/configmaps` :
      `${baseUrl}/k8s/clusters/${clusterId}/v1/configmaps`

    const res = http.post(`${url}`,
        JSON.stringify({
            "metadata": {
                "name": name,
                "namespace": namespace
            },
            "data": {"data": data}
        }),
        { cookies: cookies }
    )

    sleep(0.1)
    if (res.status != 201) {
        console.log(res)
    }
    check(res, {
        '/v1/configmaps returns status 201': (r) => r.status === 201,
    })
}

// Required params: baseurl, cookies, data, clusterId, namespace, iter
export function createSecrets(baseUrl, cookies, data, clusterId, namespace, iter)  {
    const name = `test-secrets-${iter}`

    const url = clusterId === "local"?
      `${baseUrl}/v1/secrets` :
      `${baseUrl}/k8s/clusters/${clusterId}/v1/secrets`

    const res = http.post(`${url}`,
        JSON.stringify({
            "metadata": {
                "name": name,
                "namespace": namespace
            },
            "data": {"data": data},
            "type": "opaque"
        }),
        { cookies: cookies }
    )

    sleep(0.1)
    if (res.status != 201) {
        console.log(res)
    }
    check(res, {
        '/v1/secrets returns status 201': (r) => r.status === 201,
    })
}


// Required params: baseurl, cookies, clusterId, namespace, iter
export function createDeployments(baseUrl, cookies, clusterId, namespace, iter) {
    const name = `test-deployment-${iter}`

    const url = clusterId === "local"?
      `${baseUrl}/v1/apps.deployments` :
      `${baseUrl}/k8s/clusters/${clusterId}/v1/apps.deployments`


    const res = http.post(`${url}`,
        JSON.stringify({
            "apiVersion": "apps/v1",
            "kind": "Deployment",
            "metadata": {
              "name": name,
              "namespace": namespace
            },
            "spec": {
              "selector": {
                "matchLabels": {
                  "app": name
                }
              },
              "template": {
                "metadata": {
                  "labels": {
                    "app": name
                  }
                },
                "spec": {
                  "containers": [
                    {
                      "command": ["sleep", "3600"],
                      "image": "busybox:latest",
                      "name": name
                    }
                  ],
                  "securityContext": {
                    "runAsUser": 2000,
                    "runAsGroup": 3000
                  }
                }
              }
            }
        }),
        { cookies: cookies }
    )

    sleep(0.1)
    if (res.status != 201) {
        console.log(res)
    }
    check(res, {
        '/v1/apps.deployments returns status 201': (r) => r.status === 201,
    })

}

String.prototype.toPascalCase = function toPascalCase(useSpaces = false) {
  return this
    .match(/[A-Z]{2,}(?=[A-Z][a-z]+[0-9]*|\b)|[A-Z]?[a-z]+[0-9]*|[A-Z]|[0-9]+/g)
    ?.map(x => x.charAt(0).toUpperCase() + x.slice(1).toLowerCase())
    .join(useSpaces ? ' ' : '') || '';
}

export function getPathBasename(filePath) {
  // Find the last occurrence of a path separator (either / or \)
  const lastSlashIndex = Math.max(filePath.lastIndexOf('/'), filePath.lastIndexOf('\\'));

  // If no slash is found, the entire string is the basename
  if (lastSlashIndex === -1) {
    return filePath;
  }

  // Extract the substring after the last slash
  return filePath.substring(lastSlashIndex + 1);
}
