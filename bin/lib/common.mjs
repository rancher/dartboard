import {spawnSync} from 'child_process'
import {dirname, join, relative} from 'path'
import {cwd, env} from 'process'
import {fileURLToPath} from 'url'

export const ADMIN_PASSWORD = "adminadminadmin"

export function dir(dir){
    const desiredPath = join(dirname(dirname(dirname(fileURLToPath(import.meta.url)))), dir)
    const currentPath = cwd()
    const result = relative(currentPath, desiredPath)

    return result !== "" ? result : "."
}

export function isK3d() {
    return tofuDir().endsWith("k3d")
}

export function tofuDir(){
    const default_dir = dir(join("tofu", "main", "k3d"))
    return env.TOFU_WORK_DIR ?? default_dir
}

export function tofuVar() {
    if (env.TOFU_VAR_FILE) {
        return `-var-file=${env.TOFU_VAR_FILE}`
    }
    return ""
}

export function run(cmdline, options = {}) {
    console.log(`***Running command:\n ${cmdline}\n`)
    const res = spawnSync(cmdline, [], {
        input: options.input,
        stdio: [options.input ? "pipe": "inherit", options.collectingOutput ? "pipe" : "inherit", "inherit"],
        shell: true
    })
    if (res.error){
        throw res.error
    }
    if (res.status !== 0){
        throw new Error(`Command returned status ${res.status}`)
    }
    console.log("")
    return res.stdout?.toString()
}

/** Quotes a string for Unix shell use */
export function q(s){
    if (!/[^%+,-.\/:=@_0-9A-Za-z]/.test(s)){
        return s
    }
    return `'` + s.replace(/'/g, `'"'`) + `'`
}

export function runCollectingOutput(cmdline) {
    return run(cmdline, {collectingOutput: true})
}

export function runCollectingJSONOutput(cmdline) {
    return JSON.parse(runCollectingOutput(cmdline))
}

export function sleep(s) {
    return new Promise(resolve => setTimeout(resolve, s*1000))
}

export async function retryOnError(f) {
    for (let i = 0; i < 12; i++) {
        try {
            return f()
        }
        catch (e) {
            await sleep(5)
        }
    }
}

export function helm_install(name, chart, cluster, namespace, values) {
    const json = Object.entries(values).map(([k,v]) => `${k}=${JSON.stringify(v)}`).join(",")
    run(`helm --kubeconfig=${q(cluster["kubeconfig"])} --kube-context=${q(cluster["context"])} upgrade --install --namespace=${q(namespace)} ${q(name)} ${q(chart)} --create-namespace --set-json=${q(json)}`)
}

export function getAppAddressesFor(cluster) {
    const addresses = cluster["app_addresses"]
    const loadBalancerName = guessAppFQDNFromLoadBalancer(cluster)

    // addresses meant to be resolved from the machine running OpenTofu
    // use tunnel if available, otherwise public, otherwise go through the load balancer
    const localNetworkName = addresses["tunnel"]["name"] || addresses["public"]["name"] || loadBalancerName
    const localNetworkHTTPPort = addresses["tunnel"]["http_port"] || addresses["public"]["http_port"] || 80
    const localNetworkHTTPSPort = addresses["tunnel"]["https_port"] || addresses["public"]["https_port"] || 443

    // addresses meant to be resolved from the network running clusters
    // use public if available, otherwise private if available, otherwise go through the load balancer
    const clusterNetworkName = addresses["public"]["name"] || addresses["private"]["name"] || loadBalancerName
    const clusterNetworkHTTPPort = addresses["public"]["http_port"] || addresses["private"]["http_port"] || 80
    const clusterNetworkHTTPSPort = addresses["public"]["https_port"] || addresses["private"]["https_port"] || 443

    return {
        localNetwork: {
            name: localNetworkName,
            httpURL: `http://${localNetworkName}:${localNetworkHTTPPort}`,
            httpsURL: `https://${localNetworkName}:${localNetworkHTTPSPort}`,
        },
        clusterNetwork: {
            name: clusterNetworkName,
            httpURL: `http://${clusterNetworkName}:${clusterNetworkHTTPPort}`,
            httpsURL: `https://${clusterNetworkName}:${clusterNetworkHTTPSPort}`,
        },
    }
}

export function guessAppFQDNFromLoadBalancer(cluster){
    const kf = `--kubeconfig=${cluster["kubeconfig"]}`
    const cf = `--context=${cluster["context"]}`

    return runCollectingJSONOutput(`kubectl get service --all-namespaces --output json ${q(kf)} ${q(cf)}`)
        .items
        .filter(x => x["spec"]["type"] === "LoadBalancer")
        .map(x => x["status"]["loadBalancer"]["ingress"])
        .flat()
        .filter(x => x)
        .map(x => x["ip"] + ".sslip.io" || x["hostname"])[0]
}

// imports latest built rancher and rancher-agent images into clusters
// only works for k3d at the moment
export function importImage(image, ...clusters) {
    if (isK3d()) {
        const lines = runCollectingOutput(`docker images --filter='reference=${image}' --format=json`).trim().split("\n").filter(x => x !== "")
        const images = lines.map(line => JSON.parse(line)).map(image => image.Repository + ":" + image.Tag)

        if (images.length > 0) {
            for (const cluster of clusters) {
                run(`k3d image import --cluster ${cluster["context"].replace(/^k3d-/, "")} ${images[0]}`)
            }
        }
    }
}
