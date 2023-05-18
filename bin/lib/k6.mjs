import {dir, q, run} from "./common.mjs";

const RANCHER_URL = "https://rancher.cattle-system"
const MIMIR_URL = "http://mimir.cattle-monitoring-system:9009/api/v1/push"

/** Runs k6 from inside the k8s cluster */
export function k6_run(envs, test) {
    envs["BASE_URL"] = RANCHER_URL
    const cmdline = "k6 run " + Object.entries(envs).map(([k,v]) => `-e ${q(k)}=${q(v)}`).join(" ") + " " + q(dir(test))
    console.log(`***Running equivalent of:\n ${cmdline}\n`)

    const args = Object.entries(envs).map(([k,v]) => ["-e", `${k}=${v}`]).flat()
    const overrides = {
        "apiVersion": "v1",
        "spec": {
            "containers": [
                {
                    "name": "k6",
                    "image": "grafana/k6:0.44.1",
                    "stdin": true,
                    "tty": true,
                    "args": [ "run" , args, test, "-o", "experimental-prometheus-rw"].flat(),
                    "workingDir": "/",
                    "env": [
                        {"name": "K6_PROMETHEUS_RW_SERVER_URL", "value": MIMIR_URL},
                    ],
                    "volumeMounts": [
                        {
                            "mountPath": "/k6",
                            "name": "k6-test-files"
                        },
                        {
                            "mountPath": "/k6/lib",
                            "name": "k6-lib-files"
                        }
                    ]
                },
            ],
            "volumes": [
                {
                    "name": "k6-test-files",
                    "configMap": {
                        "name": "k6-test-files"
                    }
                },
                {
                    "name": "k6-lib-files",
                    "configMap": {
                        "name": "k6-lib-files"
                    }
                }
            ]
        }
    }

    return run(`kubectl run k6 --image grafana/k6:0.44.1 --namespace=cattle-monitoring-system --rm -i --tty --restart=Never --overrides=${q(JSON.stringify(overrides))}`)
}
