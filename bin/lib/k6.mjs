import {dir, q, run} from "./common.mjs"

const MIMIR_URL = "http://mimir.tester:9009/mimir"

const K6_IMAGE = "grafana/k6:0.46.0";

/**
 * Runs a k6 script from inside the specified k8s cluster via kubectl run.
 * Specify envs, tags, and a test file from the k6/ dir (which will be transferred via ConfigMaps).
 * Set record = true for result metrics to be sent to Mimir, which is expected to be running in the same cluster.
 *
 * If the test script exercises the Kubernetes API, specify a KUBECONFIG env, the corresponding file will be transferred
 * to the cluster via a Secret.
 */
export function k6_run(cluster, envs, tags, test, record = false) {
    if (envs["KUBECONFIG"]) {
        run(`kubectl --namespace=tester delete secret kube --ignore-not-found --kubeconfig=${q(cluster["kubeconfig"])} --context=${q(cluster["context"])}`)
        run(`kubectl --namespace=tester create secret generic kube --from-file=config=${envs["KUBECONFIG"]} --kubeconfig=${q(cluster["kubeconfig"])} --context=${q(cluster["context"])}`)
        envs["KUBECONFIG"] = "/kube/config"
    }

    const envArgs = Object.entries(envs).map(([k,v]) => ["-e", `${k}=${v}`]).flat()
    const tagArgs = Object.entries(tags).map(([k,v]) => ["--tag", `${k}=${v}`]).flat()
    const outputArgs = record ? ["-o", "experimental-prometheus-rw"] : []

    const cmdline = `k6 run ${envArgs.join(" ")} ${tagArgs.join(" ")} ${q(dir(test))}`
    console.log(`***Running equivalent of:\n ${cmdline}\n`)

    const overrides = {
        "apiVersion": "v1",
        "spec": {
            "containers": [
                {
                    "name": "k6",
                    "image": K6_IMAGE,
                    "stdin": true,
                    "tty": true,
                    "args": [ "run" , envArgs, tagArgs, test, outputArgs].flat(),
                    "workingDir": "/",
                    "env": [
                        {"name": "K6_PROMETHEUS_RW_SERVER_URL", "value": MIMIR_URL + "/api/v1/push"},
                        {"name": "K6_PROMETHEUS_RW_TREND_AS_NATIVE_HISTOGRAM", "value": "true"},
                        {"name": "K6_PROMETHEUS_RW_STALE_MARKERS", "value": "true"},
                    ],
                    "volumeMounts": [
                        { "mountPath": "/k6", "name": "k6-test-files" },
                        { "mountPath": "/k6/lib", "name": "k6-lib-files"}
                    ].concat(envs["KUBECONFIG"] ? [{ "mountPath": "/kube", "name": "kube" }]: [])
                },
            ],
            "volumes": [
                { "name": "k6-test-files", "configMap": { "name": "k6-test-files" } },
                { "name": "k6-lib-files", "configMap": { "name": "k6-lib-files" } }
            ].concat(envs["KUBECONFIG"] ? [ { "name": "kube", "secret": { "secretName": "kube" } } ] : [])
        }
    }

    return run(`kubectl run k6 --image ${K6_IMAGE} --namespace=tester --rm -i --tty --restart=Never --overrides=${q(JSON.stringify(overrides))} --kubeconfig=${q(cluster["kubeconfig"])} --context=${q(cluster["context"])}`)
}
