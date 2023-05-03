import { spawnSync } from 'child_process'
import { dirname, relative, join } from 'path'
import { cwd } from 'process'
import { fileURLToPath } from 'url'

export const ADMIN_PASSWORD = "adminadminadmin"

export function dir(dir){
    const desiredPath = join(dirname(dirname(dirname(fileURLToPath(import.meta.url)))), dir)
    const currentPath = cwd()
    const result = relative(currentPath, desiredPath)

    return result !== "" ? result : "."
}

export function run(cmdline, options = {}) {
    console.log(`***Running command:\n ${cmdline.replaceAll(",", "\,")}\n`)
    const cmd = cmdline.split(" ")[0]
    const args = cmdline.split(" ").slice(1)
    const res = spawnSync(cmd, args, {
        input: options.input,
        stdio: [options.input ? "pipe": "inherit", options.collectingOutput ? "pipe" : "inherit", "inherit"],
        shell: false
    })
    if (res.error){
        throw res.error
    }
    if (res.status !== 0){
        throw new Error(`Command returned status ${res.status}: ${cmdline.replaceAll(",", "\\,")}`)
    }
    console.log("")
    return res.stdout?.toString()
}

export function runCollectingOutput(cmdline) {
    return run(cmdline, {collectingOutput: true})
}

export function runCollectingJSONOutput(cmdline) {
    return JSON.parse(runCollectingOutput(cmdline))
}

export function sleep(s) {
    return new Promise(resolve => setTimeout(resolve, s*1000));
}
