# 2025-07-09 - Setup Fleet Standalone With 5000 K3k vClusters

## Setup

The setup uses AWS. It needs two 53-node RKE2 clusters for k3k (downstream) and one 4-node cluster (upstream) for Fleet itself.
The clusters always use three control plane nodes, the remaining nodes are agents.

AWS instance types:
* c5d.12xlarge for fleet cluster (upstream)
* c5d.9xlarge for k3k clusters (downstream)

### Dartboard

Several changes were done to dartboard to enable this test:

* added [docker.io mirror](https://github.com/rancher/dartboard/pull/90)
* install [rke2 from prime ribs](https://github.com/rancher/dartboard/pull/87)
* etcd quota increase in rke2 install script
* max_pods fix
* allow [more SSH connections to bastion host](https://github.com/rancher/dartboard/pull/89)
* configured node_cidr_mask to /22
* configured [larger cidr blocks /20](https://github.com/rancher/dartboard/pull/88)

### Scripts

For reference here are some of the scripts used in this experiment:
* [create-vclusters.sh](https://github.com/rancher/dartboard/blob/20250709_fleet_standalone_5000_clusters_scripts/docs/20250709%20-%20Fleet%20Standalone%205000%20Clusters/create-vclusters.sh)
* [label-empty-nodes.sh](https://github.com/rancher/dartboard/blob/20250709_fleet_standalone_5000_clusters_scripts/docs/20250709%20-%20Fleet%20Standalone%205000%20Clusters/label-empty-nodes.sh)
* [create-kubeconfigs.sh](https://github.com/rancher/dartboard/blob/20250709_fleet_standalone_5000_clusters_scripts/docs/20250709%20-%20Fleet%20Standalone%205000%20Clusters/create-kubeconfigs.sh)
* [register-agent.sh](https://github.com/rancher/dartboard/blob/20250709_fleet_standalone_5000_clusters_scripts/docs/20250709%20-%20Fleet%20Standalone%205000%20Clusters/register-agent.sh)
* [setup-fleet-release.sh](https://github.com/rancher/dartboard/blob/20250709_fleet_standalone_5000_clusters_scripts/docs/20250709%20-%20Fleet%20Standalone%205000%20Clusters/setup-fleet-release.sh)


### K3K

This step is repeated in parallel on the second 53-node cluster.

K3K needs storage:

```
kubectl apply -f https://raw.githubusercontent.com/rancher/local-path-provisioner/v0.0.31/deploy/local-path-storage.yaml
kubectl patch cm -n local-path-storage local-path-config --type merge \
  --patch '{"data":{"config.json": "{\"nodePathMap\":[{\"node\":\"DEFAULT_PATH_FOR_NON_LISTED_NODES\",\"paths\":[\"/data/storage\"]}]}"}}'
kubectl patch storageclass local-path -p '{"metadata": {"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'
```

Afterwards a default k3k installation is created: `helm install --namespace k3k-system --create-namespace k3k k3k/k3k`

To allow for the incremental creation of vClusters, or to recover a failed attempt, only labeled agent nodes are used.
See `label-empty-nodes.sh`, which will label all agent nodes with less than 150 pods with `wave=1`.

Afterwards the vclusters can be created with `./create-vclusters.sh 1 2500 1`. This will create vClusters 1 to 2500, on nodes labeled with `wave=1`.

Each vcluster is exposed by a node port. There are enough ports for 2500 clusters.

The clusters are configured:
* expose API via node port
* no exposed etcd ports, to avoid running out of ports
* shared mode, as virtual mode uses too many resources
* nodeSelector, to use a single agent node
* TLS SAN is set to `<random control plane node>.ec2.internal`

With the `c5d.9xlarge` instance type, this allows for ~50 vClusters per node. However, there is not a lot of headroom to run applications on these clusters. K3s pods will crash and respawn if there are not enough resources, or they run into timeouts.

The script creates 50 clusters at the same time and waits for the last cluster to be ready, so each node is not overloaded. This can probably be increased to fasten this step, but using too much CPU will crash k3s, which leads to cascading failures.

### Fleet

`setup-fleet-release.sh` installs Fleet without the local cluster and increased worker counts.

* labels the agent node in upstream
* use `nodeSelector.role=agent` to run fleet-controller on this node


#### Register Agents

We switch to the bastion host and install the agents from there. This is faster and the network setup (hostnames and api ports) is easier.

1. Copy the kubeconfigs (upstream, downstream-0-0, downstream-0-1) and modify their server to point to one of the control plane nodes. The AWS setup does not include a loadbalancer for the API, so we pick one at random.
```
node=$(kubectl get nodes --selector='node-role.kubernetes.io/control-plane' -o jsonpath="{.items[*].status.addresses[?(@.type=='Hostname')].address}" | tr ' ' '\n' | head -1 )
sed -e 's@"server": "https://.*@"server": "https://'$node.ec2.internal':6443"@' default_config/downstream-0-0.yaml > default_config/downstream-external-0.yaml
```

2. Copy the necessary scripts and install tooling

* create-kubeconfigs.sh, register-agent.sh
* k3kcli, kubernetes-clients

3. Export the kubeconfigs, so we can use contexts

```
export KUBECONFIG=$PWD/upstream.yaml:$PWD:downstream-0-0.yaml:$PWD:downstream-0-1.yaml
```

4. Create kubeconfigs for all 5000 k3k clusters using the k3kcli

```
kubectl config use-context downstream-0-0
for ns in $(kubectl get namespaces --no-headers -ocustom-columns=:metadata.name | grep k3k-downstream); do
    ./create-kubeconfigs.sh "$ns";
done;
```

Repeat for downstream-0-1.

5. Register the agents

```
kubectl config use-context upstream
./register-agent.sh
```

Agents are registered with manager-initiated registration.

* create a kubeconfig secret for the vcluster on upstream
* create a cluster referencing the secret on upstream

The script does not create all registrations at the same time, but pauses every few hundred clusters. However, it also does not wait for cluster readieness. The agentmanagement seems robust, it will keep processing clusters without crashing.
Failure could be indicated by missing cluster namespaces, which can be fixed by adding a label to force a cluster reconcile.
