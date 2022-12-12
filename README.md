# Scalability tests

This repo collects code, instructions and results for scalability tests on the Rancher product family.

## Usage

```
cd terraform
terraform init
terraform apply -auto-approve
```

npx cypress open --config watchForFileChanges=false

## Troubleshooting

### Kubernetes cluster unreachable

If you get this error from `terraform apply`:
```
│ Error: Kubernetes cluster unreachable: Get "https://upstream.local.gd:6443/version": dial tcp 127.0.0.1:6443: connect: connection refused
```

SSH tunnels might be broken. Reopen them via:
```shell
./config/open-tunnels-to-upstream-*.sh
```

### k3d: hard destroy and recreate

- Hard destroy everything (if Terraform fails): `rm terraform.tfstate ; k3d cluster delete --all ; docker network rm k3d`
- Hard recreate everything from scratch:

```sh
rm terraform.tfstate ; k3d cluster delete --all ; docker container ls --format '{{.Names}}' | grep kine | xargs -n1 docker kill ; docker container ls --all --format '{{.Names}}' | grep kine | xargs -n1 docker rm ; docker network ls --format '{{.ID}} {{.Name}}' | grep k3d | awk '{print $1}' | xargs -n1 docker network rm ; terraform init; terraform apply -auto-approve
```
