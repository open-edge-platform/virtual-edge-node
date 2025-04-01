# Edge Node Simulator - For Testing Purposes Only

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Get Started](#get-started)
- [Contribute](#contribute)
- [Integration Tests](#integration-tests)

## Overview

This sub-repository contains the implementation of the Edge Node Simulator (EN-SIM) for Edge Infrastructure Manager.

Additionally, the repo contains the Edge Infrastructure Manager End-to-End Integration Tests
which are used as sanity tests to evaluate a new release.

## Features

This sub-repository is designed to provide mechanisms to experiment with Infrastructure Manager
by simulating edge nodes.
A simulated edge node performs the same calls to the Infrastructure Manager southbound
interfaces as an actual edge node would do, for the cases of:

- Day 0: onboarding (registering the edge node) and provisioning (installing OS);
- Day 1: execution of Infrastructure Manager agents (node, update, hardware-discovery, telemetry);
- Day 2: schedule of maintenance mode.

With a simulated edge node it is possible to exercise day 0, day 1 and day 2 tests with Infrastructure Manager.
While by using the simulator, it is possible to run those same tests at scale
(e.g., thousands of simulated edge nodes).
Thus, edge node simulator can be used to easily prototype new features in Infrastructure Manager
and validate them, even at scale.

Notice: the edge-node-simulator (ensim) is meant to simulate workloads for Infrastructure Manager only.

## Get Started

Instructions on how to install and set up the Edge Node Simulator on your development machine.

This sub-repository has a server and a client cmds written to do so, placed at `cmd/ensim/`.
The server defines where/how the `ensim` instances are going to be simulated,
while the client is just a simple CLI interface to communicate with the server.

### Dependencies

Firstly, please verify that all dependencies have been installed.

```bash
# Return errors if any dependency is missing
make dependency-check
```

This code requires the following tools to be installed on your development machine:

- [Go\* programming language](https://go.dev) - check [$GOVERSION_REQ](../version.mk)
- [golangci-lint](https://github.com/golangci/golangci-lint) - check [$GOLINTVERSION_REQ](../version.mk)
- [go-junit-report](https://github.com/jstemmer/go-junit-report) - check [$GOJUNITREPORTVERSION_REQ](../version.mk)
- [gocover-cobertura](github.com/boumenot/gocover-cobertura) - check [$GOCOBERTURAVERSION_REQ](../version.mk)
- [buf](https://github.com/bufbuild/buf) - check [$BUFVERSION_REQ](../version.mk)
- [protoc-gen-doc](https://github.com/pseudomuto/protoc-gen-doc) - check [$PROTOCGENDOCVERSION_REQ](../version.mk)
- [protoc-gen-go](https://pkg.go.dev/google.golang.org/protobuf) - check [$PROTOCGENGOVERSION_REQ](../version.mk)
- [protoc-gen-go-grpc](https://pkg.go.dev/google.golang.org/grpc) - check [$PROTOCGENGOGRPCVERSION_REQ](../version.mk)

You can install Go dependencies by running `make go-dependency`.

### Requirement

Both ensim and ensim charts require the CA certificate of the target orchestrator,
this one can be obtained in the target cluster using (TODO: add example from orch-infra namespace).
Then save it and apply the secret in the orchestrator cluster where ensim is going to be deployed.
Edit the namespace keyword in the secret, make sure it matches the namespace where you are going to apply it.

```bash
# create the enic namespace if it does not exist
kubectl create ns ensim

kubectl get secret -n orch-gateway tls-orch  -o yaml | grep -v '^\s*namespace:\s' > cert.yaml
sed -i'' -e "s|name: .*|name: tls-orch|" cert.yaml

kubectl apply --namespace=ensim -f cert.yaml
```

Update the values for \<onb-user\>,  \<orch-user\>, \<onb-pass\> & \<orch-pass\> whereas
 \<api-pass\> and \<api-user\> are not mandatory

Notice: if running the binary directly, extract the CA certificate from the cert.yaml and save it to a file.

### Build the Binary

Build the project as follows:

```bash
# Build go binary
make build
```

The binary is installed in the [$OUT_DIR](../common.mk) folder.

### Usage

> NOTE: This guide shows how to deploy EN-SIM for local development or testing only.

To run the server, make sure the following information is provided: orchestrator FQDN, orchestrator CA path.

```bash
./out/ensim/server/main -globalLogLevel info  -orchCAPath orch-ca.crt -orchFQDN kind.internal
```

Make sure that where your server is executed the orchFQDN DNS name is properly configured in the `/etc/hosts` file.
The option `globalLogLevel` can be specified as `info` or `debug`.
The server is going to start a gRPC server on port `5001`. To connect to the server run the client as:

```bash
./out/ensim/client/main -addressSimulator localhost:5001
```

The server and client don't need to run in the same machine.
The client CLI is intuitive, as long as the server is running,
the client can stop/disconnect and connect again, it is stateless.

## Build ensim - docker

To build ensim docker container with server and client run:

```bash
make docker-build
```

If using kind cluster, make sure to load the imaget into it.

```bash
make kind-load
```

## Helm

To deploy the en-sim chart, for example execute the following command
(check and fill the orchIP and orchFQDN params).

```bash
helm upgrade --install en-sim ./charts/ensim/ -n scale -f ensim-values.yaml
```

The set of ensim parameters that can be defined in `ensim-values.yaml` are shown below:

```yaml
configArgs:
  server:
    gRPCPort: 3196 # The port of the ensim gRPC server address.
    globalLogLevel: "info" # or "debug" to set the log level.
    oamServerAddress: "0.0.0.0:2379" # server address for the OAM, k8s ready/liveness server.
    orchCAPath: "/usr/local/share/ca-certificates/orch-ca.crt" # Filepath of CA certificate to access the orchestrator cluster.
    orchFQDN: "kind.internal" # The orchestrator FQDN.
    orchIP: "" # The orchestrator IP address - to be specified if FQDN is not in the /etc/hosts of the host machine.
```

## Run ensim client CLI

To execute the ensim CLI client run the following command:

```bash
./out/ensim/client/main -addressSimulator localhost:3196 -projectID 8e202529-f980-4bc2-9e34-e2e927336d36 -apiUser adminTestUser1 -apiPass adminTestPassword1! -onbUser adminTestUser1 -onbPass adminTestPassword1!
```

The parameters are the following:

```bash
addressSimulator: localhost:3196 # The gRPC address (server:port) of the ensim server.
onbUser: "<onb-user>" # Keycloak username to perform onboarding/provisioning of ensim.
onbPass: "<onb-pass>" # Password for the onbUser.
apiUser: "<api-user>" # Keycloak username to perform teardown of ensim.
apiPass: "<api-pass>" # Password for the apiUser.
projectID: "" # Project UUID to be used by ensim to retrieve keycloak 
              # credentials to access orchestrator interfaces.
```

These flags are not mandatory, with the exception of `addressSimulator`.
But they facilitate the interaction with the ensim client CLI,
as the flag parameters are offered as default settings for the CLI arguments.

## Contribute

The folders simulator (sim) and edge-node (en) source code are decoupled from each other,
allowing modular changes in code in both of them.
In en folder, the sub-folders are organized by the main procedures of an edge node:

- onboarding: contains onboarding (interactive or not) and provisioning related source code;
- keycloak: contains token refresh manager routines and methods to perform the retrieval of the client token;
- agents: contains all the simulation of Edge Infrastructure Manager agents calls.

And in ensim, even simpler, the code base is in a single folder containing mainly the ensim store,
the northbound gRPC server implementation, and the cli client code base.

All the source code is verified with golang linters, so a minimum code format/quality is maintained in the project.
Make sure to run `make lint` when contributing to the project.

As the code base and call flows are complex, the components is not maintained with unit tests,
but integration tests perform all the verifications needed to validate the functionalities
needed to exercise Edge Infrastructure Manager interfaces.
And those integration tests are meant to be used as quality gates for Edge Infrastructure Manager releases.

All source code of ensim is maintained and released in CI using the VERSION file.
In addition, the chart of ensim is versioned in the same tag with the VERSION file, this is mandatory
to keep all charts versions and app versions compatible in the same repository.
It is just a hack to facilitate the release process of them, given the source code is so related.

Make sure the versions in ensim go.mod match those of the components of the
Edge Infrastructure Manager release.
That's the only manner to maintain ensim compatible with Edge Infrastructure Manager.
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

## Integration Tests

The edge node simulator has a set of integration tests that exemplify its functionalities.
These are contained inside the `test/infra` folder, and are meant to provide examples of
day 0/1/2 tests for Edge Infrastructure Manager.

Other tests in `test/ensim` folder are meant to run a smoke test of EN simulator functionalities, and
retrieve a summary of the status of its simulated edge nodes.

About the integration tests, they are written using the framework `ginkgo` and are organized as follows:

- day 0: realizes 2 test cases for Edge Infrastructure Manager,
onboarding/provisioning with interactive and non-interactive mechanisms;
- day 1: realizes 2 test cases for agents in Edge Infrastructure Manager,
one by checking if those are stable/running for a given period of time,
and the other by checking if Edge Infrastructure Manager can identify as
connection lost an edge node in case its agents stop operating.
- day 2: realizes 4 test cases for Edge Infrastructure Manager related
to maintenance schedules in 4 different granuralities, by host, by site,
by region and by root region.

All the tests exercise the Edge Infrastructure Manager functionalities exercising
northbound and southbound interfaces of Edge Infrastructure Manager, treating it as a
unique Device Under Test (DUT).
The tests are parameterized and can be executed by CI pipelines modularly.
