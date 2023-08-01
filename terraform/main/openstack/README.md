# Openstack provider

## Opensack Requirements

- Openstack with following features enabled:
  - Router
  - Floating IP (with internet connectivity IP pool address)
    - One IP is needed for each clusters (upstream/downstream/tester) and one for the bastion
- An OpenRC file (Openstack credentials)

## Configuration

- An External Network (Gateway and FloatingIP will be attached from this network)
- A private Network (the Subnet will spawn in this Network)

## Not yet implemented

- No multi K3S masters
  - No shared databases deployed
  - No Octavia (loadbalancer) deployed in front of masters (floating IP is attached to the first/master node)

## Design

- Create a single private subnet in the provided Openstack Network. All instances will be spawn in this subnet.
- An Openstack Router is spawn to allow the usage of Floating IPs (from Ext-Net) and allow SNAT
- A floating IP is attached to the bastion and to each K3S Master servers

## Usage

```shell
source ./openrc.bash # Openstack Credentials
vim input.tf         # Tweak your infra
terraform apply      # Deploy your infra
```
