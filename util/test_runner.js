const { spawnSync } = require('node:child_process')
const { dirname, join } = require('node:path')
const { chdir } = require('node:process')

function run(cmd, args = []) {
    const cmdline = `${cmd} ${args.join(" ")}`
    console.log(`***Running: ${cmdline}`)
    const res = spawnSync(cmd, args, {
        stdio: ["inherit", "inherit", "inherit"],
        shell: cmd.includes(" ")
    })
    if (res.status !== 0){
        throw new Error(`Command returned status ${res.status}: ${cmdline}`)
    }
    return res.stdout || ""
}

function cd(dir){
    chdir(join(dirname(__dirname), dir))
}

// Terraform: clean up
cd("terraform")
run("terraform init")
// HACK: Helm deployer does not always clean up successfully. Get rid of its state, cluster is being recreated anyway
const states = run("terraform state list").split("\n")
for (const state in states) {
    if (state.indexOf("helm_release") > 0){
        run("terraform", ["state", "rm", state])
    }
}
run("terraform destroy -auto-approve")

// Terraform: apply
run("terraform apply -auto-approve")
const params = JSON.parse(run("terraform output -json"))
console.log(params)

// k6: set Rancher up
cd("k6")
const ADMIN_PASSWORD = "adminadminadmin"
run("k6", [
    "run",
    "-e", `BASE_URL=${params["base_url"]}`,
    "-e", `BOOTSTRAP_PASSWORD=${params["bootstrap_password"]}`,
    "-e", `PASSWORD=${ADMIN_PASSWORD}`,
    "-e", `IMPORTED_CLUSTER_NAMES=${params.downstream_cluster_names.join(",")}`,
    "./rancher_setup.js"
])
