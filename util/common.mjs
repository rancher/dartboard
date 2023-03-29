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
        stdio: ["inherit", "pipe", "inherit"],
    })
    if (res.error){
        throw res.error
    }
    const output = res.stdout.toString()
    console.log(output)
    if (res.status !== 0){
        throw new Error(`Command returned status ${res.status}: ${cmdline}`)
    }
    return output
}
