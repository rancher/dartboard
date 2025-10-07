import {check, sleep} from 'k6';
import http from 'k6/http';

// Create Config Maps
// Required params: baseurl, coookies, data, cluster, namespace, iter
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


// Create Secrets
// Required params: baseurl, coookies, data, cluster, namespace, iter
export function createSecrets(baseUrl, cookies, data, clusterId, namespace, iter)  {
    const name = `test-secret-${iter}`

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

// Create Secrets with Labels
// Required params: baseurl, coookies, data, cluster, namespace, iter
export function createSecretsWithLabels(baseUrl, cookies, data, clusterId, namespace, iter)  {
    const name = `test-secret-${iter}`
    const key_1 = "cow"
    const value = "geeko"

    const url = clusterId === "local"?
      `${baseUrl}/v1/secrets` :
      `${baseUrl}/k8s/clusters/${clusterId}/v1/secrets`

    const res = http.post(`${url}`,
        JSON.stringify({
            "metadata": {
                "name": name,
                "namespace": namespace,
                "labels": {[key_1]:value}
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
// Required params: baseurl, coookies, cluster, namespace, iter
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


export function createStorageClasses(baseUrl, cookies, namespace, iter){

    const name = `test-storage-class-${iter}`

    const create = http.post(`${baseUrl}/v1/storage.k8s.io.storageclasses`, JSON.stringify({
        "type": "storage.k8s.io.storageclass",
        "metadata": {
            "name": name,
            "namespace": namespace
        },
        "provisioner": "driver.longhorn.io",
        }),
        {cookies: cookies}
    )

    sleep(0.1)
    if (res.status != 201) {
        console.log(res)
    }
    check(create, {
        '/v1/storage.k8s.io.storageclasses returns status 201': (r) => r.status === 201,
    })

 }