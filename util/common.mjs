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
    console.log(`***Running command:\n ${cmdline}\n`)
    const res = spawnSync(cmd, args, {
        stdio: "inherit",
    })
    if (res.error){
        throw res.error
    }
    if (res.status !== 0){
        throw new Error(`Command returned status ${res.status}: ${cmdline}`)
    }
}

export function runWithInput(input, cmd, ...args) {
    const cmdline = `${cmd} ${args.join(" ")}`
    console.log(`***Running command:\n ${cmdline}\n`)
    const res = spawnSync(cmd, args, {
        input: input,
        stdio: ["pipe", "inherit", "inherit"],
    })
    if (res.error){
        throw res.error
    }
    if (res.status !== 0){
        throw new Error(`Command returned status ${res.status}: ${cmdline}`)
    }
}

export function runWithOutput(cmd, ...args) {
    const cmdline = `${cmd} ${args.join(" ")}`
    console.log(`***Running command:\n ${cmdline}\n`)
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

export function runWithJsonOutput(cmd, ...args) {
    const output = runWithOutput(cmd, ...args)
    return JSON.parse(output)
}
