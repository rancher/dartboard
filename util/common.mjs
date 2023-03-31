import { spawnSync } from 'child_process'
import { dirname, join } from 'path'
import { chdir } from 'process'
import { fileURLToPath } from 'url'

export const ADMIN_PASSWORD = "adminadminadmin"

export function cd(dir){
    const __filename = fileURLToPath(import.meta.url);
    const __dirname = dirname(dirname(__filename))

    chdir(join(__dirname, dir))
}

export function run(cmdline, options = {}) {
    console.log(`***Running command:\n ${cmdline}\n`)
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
        throw new Error(`Command returned status ${res.status}: ${cmdline}`)
    }
    return res.stdout?.toString()
}

export function runCollectingOutput(cmdline) {
    return run(cmdline, {collectingOutput: true})
}

export function runCollectingJSONOutput(cmdline) {
    return JSON.parse(runCollectingOutput(cmdline))
}
