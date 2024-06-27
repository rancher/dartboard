# OpenStack provider

## OpenStack Requirements
- OpenStack with following features enabled:
  - Router
  - Floating IPs (with internet connectivity IP pool address)
    - One IP is needed for each cluster (upstream/downstream/tester) and one for the bastion
- An OpenRC file (OpenStack credentials)
- An External Network ID (Gateway and Floating IPs will be attached to this network)
- Optionally, a Private Network ID

## Design
- A Private Network is either specified via ID (see above) or crated
- A single Subnet is created in the Private Network. All instances will be connected in this subnet
- An OpenStack Router is spawned to allow the usage of Floating IPs (from the External Network) and allow SNAT
- A Floating IP is attached to the bastion host and to the first Server of each K3S cluster

## Not yet implemented
Octavia (loadbalancer) server nodes. Currently, floating IPs are attached to the first server node only.

This set of OpenTofu files has been tested so far on the OVHcloud OpenStack implementation.

## Usage

Deployment only:
```shell
source ./openrc.bash # OpenStack Credentials
vim input.tf         # Tweak parameters
tofu apply           # Deploy (Tofu only). ./bin/setup.mjs can be used to set up the test suite
```
