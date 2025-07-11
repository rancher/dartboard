import {check, sleep} from 'k6';
import http from 'k6/http';
import { randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

// Create Config Maps
// Required params: iter, name, namespace, baseurl, data, cookies
export function createConfigMaps(baseUrl, cookies, cluster, namespace, data, iter) {
    const name = `test-config-map-${iter}`

    const url = cluster === "local"?
      `${baseUrl}/v1/configmaps` :
      `${baseUrl}/k8s/clusters/${cluster}/v1/configmaps`

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


// Create Secrets
// Required params: iter, name, namespace, baseurl, data
export function createSecrets(baseUrl, cookies, cluster, namespace, data, iter)  {
    const name = `test-secrets-${iter}`

    const url = cluster === "local"?
      `${baseUrl}/v1/secrets` :
      `${baseUrl}/k8s/clusters/${cluster}/v1/secrets`

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


// Create Deployments 
// Required params: iter, name, namespace, baseurl, cookies
export function createDeployments(baseUrl, cookies, cluster, namespace, iter) {
    const name = `test-deployment-${iter}`

    const url = cluster === "local"?
      `${baseUrl}/v1/apps.deployments` :
      `${baseUrl}/k8s/clusters/${cluster}/v1/apps.deployments`


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
                      "command": [
                        "bash",
                        "-c",
                        `echo test; sleep ${randomIntBetween(10,60)}; exit 1;`
                      ],
                      "image": "ubuntu",
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
