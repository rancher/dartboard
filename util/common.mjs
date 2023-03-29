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

export function run(cmd, ...args) {
    const cmdline = `${cmd} ${args.join(" ")}`
    console.log(`***Running: ${cmdline}`)
    const res = spawnSync(cmd, args, {
        stdio: ["inherit", "inherit", "inherit"],
    })
    if (res.error){
        throw res.error
    }
    if (res.status !== 0){
        throw new Error(`Command returned status ${res.status}: ${cmdline}`)
    }
}

export function runCollectingOutput(cmd, ...args) {
    const res = spawnSync(cmd, args, {
        stdio: ["ignore", "pipe", "inherit"],
    })
    if (res.error){
        throw res.error
    }
    if (res.status !== 0){
        throw new Error(`Command returned status ${res.status}: ${cmd} ${args.join(" ")}`)
    }
    return res.stdout.toString()
}
