# Edge Node in a Container - For Testing Purposes Only

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Get Started](#get-started)
- [Contribute](#contribute)

## Overview

This sub-repository contains the implementation of the Edge Node in a Container (ENiC).

ENiC (Edge Node in a Container) is a lightweight implementation of an edge node.
It performs the processes of onboarding and provisioning using simulated
interfaces with the orchestrator.
And it installs and run the actual Bare Metal Agents (BMAs) inside the container,
enabling the execution of infrastructure, cluster and application use cases.

## Features

First and foremost, notice:

- ENiC is not part of EMF;
- ENiC is developed and released for testing purposes only;
- ENiC can be used to validate EMF features, in infrastructure,
  cluster and application scopes;
- ENiC is used by the Continuous Integration pipeline of EMF to
  run integration tests in the scope of infrastructure, cluster,
  application and UI domains;
- ENiC does not contain external (management/control) interfaces,
  it only communicates with an EMF orchestrator for the sole purpose
  of testing its functionalities;
- ENiC performs the onboarding/provisioning process of an actual edge node
  using simulated calls, which exercise the same interfaces as an actual
  edge node does in the EMF orchestrator;
- ENiC enables the execution of Bare Metal Agents in the same manner as an
  actual edge node, i.e., BMAs are installed (from their .deb packages),
  configured and executed as systemd services;
- ENiC requires its container to run in privileged mode and with root
  user because it needs to install the BMAs after it is initiated,
  and some BMAs require access to the “dmidecode” tool,
  used to retrieve the system UUID and Serial Number;
- Further hardening of ENiC is going to be performed to reduce the
  privileges/capabilities of the container and possibly avoid the execution
  of it using the root user. This requires further investigation.

## Get Started

### Prereqs

- Golang
- Docker
- update the values for \<password\> , \<coder-ip\> , \<api-user\> & \<onb-user\>

### Download BMA (Bare Metal Agents) from the Release Service

You can download the BMA from the release service by:
Clone this repo and then execute the steps below:

```shell
make bma_packages   # Downloads BMA packages
make bma_versions   # Applies the BMA versions into the chart values.
make apply-version  # Applies the VERSION into the chart versions.
```

### Build and load the container image

This repository also contains the Dockerfile and Helm Chart required to
emulate the EN on top of Kubernetes.

To build the container(s):

```shell
make docker-build
```

This builds two different containers:

- `enic` from [./docker/Dockerfile](./docker/Dockerfile)
- `enic-utils` from [./docker/Dockerfile.utils](. /docker/Dockerfile.utils)

If you are on the orchestrator machine with kind cluster, run the following to load the
images built with BMA packages:

```shell
make kind-load
```

If your build machine is different from the orchestrator machine, you will need to
export the docker images & import them to kind.

```shell
make image-export
```

This should create 2 tar files edge-node.tar.gz and edge-node-utils.tar.gz. Copy the
files to your kind cluster and import them:

```shell
kind load image-archive edge-node.tar.gz
kind load image-archive edge-node-utils.tar.gz
docker exec kind-control-plane crictl images | grep enic
```

Both of these containers are deployed as part of the same POD, but they have
very different roles:

### Edge Node Logs

Contains the BMA and runs under `systemd`. Emulates the Edge Node.
Once it starts it will set up all the required configuration for the BMA and
install them (the Debians are built-in the container).

BMAs are installed at runtime to let the system use the definitive configuration;
this has been done to streamline the build process. Otherwise many configs needed
to be changed to use the right runtime parameters (diff between build and deploy time).

To see how the onboarding/provisioning is proceeding you can

```shell
kubectl -n enic exec -it $(kubectl -n enic get pods -l app=enic --no-headers | awk '{print $1}') -c edge-node -- journalctl -u onboard -f
```

To see how the installation is proceeding you can

```shell
kubectl -n enic exec -it $(kubectl -n enic get pods -l app=enic --no-headers | awk '{print $1}') -c edge-node -- journalctl -u agents -f
```

To get useful information out of the agent logs you can use commands like:

```shell
kubectl -n enic exec -it $(kubectl -n enic get pods -l app=enic --no-headers | awk '{print $1}') -c edge-node -- journalctl -u cluster-agent -f
kubectl -n enic exec -it $(kubectl -n enic get pods -l app=enic --no-headers | awk '{print $1}') -c edge-node -- journalctl -u node-agent -f
```

(or any of the other agents)

If the cluster is stuck in pending, you can check the service that
overrides the DNS entries in rancher and fleet agents:

```shell
kubectl -n enic exec -it $(kubectl -n enic get pods -l app=enic --no-headers | awk '{print $1}') -c edge-node -- journalctl -u rancher-system-agent -f
kubectl -n enic exec -it $(kubectl -n enic get pods -l app=enic --no-headers | awk '{print $1}') -c edge-node -- journalctl -u rke2-server -f
kubectl -n enic exec -it $(kubectl -n enic get pods -l app=enic --no-headers | awk '{print $1}') -c edge-node -- journalctl -u cluster-dns -f
```

### Edge Node Scripts

Contains the utility scripts defined in [./scripts](./scripts)
which are responsible to run the `onboard` and `agents` systemd services.

### Deploy

Until the images are published use `make kind-load`

The ENiC depends on having access to the Orchestrator CA certificate, and it
expects to find such certificate in a secret named `tls-orch` in the same
namespace the chart is installed.

To create such certificate you use the following commands:

```shell

# create the enic namespace if it does not exist
kubectl create ns enic

kubectl get secret -n orch-gateway tls-orch  -o yaml | grep -v '^\s*namespace:\s' > cert.yaml
sed -i'' -e "s|name: .*|name: tls-orch|" cert.yaml
kubectl apply --namespace=enic -f cert.yaml
```

```shell
helm upgrade --install enic -n enic ./chart/ \
  --set param.orch_fqdn=<cluster-fqdn> \
  --set param.orch_ip=<coder-ip> \
  --set param.orchUser=<onb-user> \
  --set param.orchPass=<password> \
  --set global.registry.name=<registry> \
  --set param.debug=true
```

## Contribute

All source code of ENiC is maintained and released in CI using the VERSION file.
In addition, the chart of ENiC is versioned in the same tag with the VERSION file, this is mandatory
to keep all charts versions and app versions compatible in the same repository.
It is just a hack to facilitate the release process of them, given the source code is so related.

Make sure the versions in ENiC go.mod match those of the components of the
Edge Infrastructure Manager release.
That's the only manner to maintain ENiC compatible with Edge Infrastructure Manager.
For example, update the go.mod with:

```yaml
github.com/open-edge-platform/infra-core/inventory/v2 v2.9.0
```

Run the go.mod/go.sum update with:

```bash
make go-tidy
```

See the [docs](docs) for advanced architectural details:

- [Architecture and Workflows](docs/internals.md)
