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
export function createSecretsWithLabels(baseUrl, cookies, data, clusterId, namespace, iter, key, value)  {
    const name = `test-secret-${iter}`
    const key_1 = key
    const value_1 = value

    const url = clusterId === "local"?
      `${baseUrl}/v1/secrets` :
      `${baseUrl}/k8s/clusters/${clusterId}/v1/secrets`

    const res = http.post(`${url}`,
        JSON.stringify({
            "metadata": {
                "name": name,
                "namespace": namespace,
                "labels": {[key_1]:value_1}
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


export function createStorageClasses(baseUrl, cookies, clusterId, iter){

    const name = `test-storage-class-${iter}`

    const url = clusterId === "local"?
      `${baseUrl}/v1/storage.k8s.io.storageclasses` :
      `${baseUrl}/k8s/clusters/${clusterId}/v1/storage.k8s.io.storageclasses`


    const res = http.post( `${url}`, JSON.stringify({
        "type": "storage.k8s.io.storageclass",
        "metadata": {"name": name},
        "parameters": { "numberOfReplicas": "3", "staleReplicaTimeout": "2880" },
        "provisioner": "driver.longhorn.io",
        "allowVolumeExpansion": true,
        "reclaimPolicy": "Delete",
        "volumeBindingMode": "Immediate"
        }),
        {cookies: cookies}
    )

    sleep(0.1)
    if (res.status != 201) {
        console.log(res)
    }
    check(res, {
        '/v1/storage.k8s.io.storageclasses returns status 201': (r) => r.status === 201,
    })
}
