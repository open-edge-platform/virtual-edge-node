# Integration Tests

The following steps provide guidance on how to run the integration tests for days 0, 1 and 2 using
the Edge Node simulator.
All test cases have descriptions in their respective files/definitions.

## Requirements

For all the day0, day1 and day2 tests the following environment variables need to be defined.

```bash
ORCH_FQDN="" # The FQDN of the target orchestrator cluster
ENSIM_ADDR="localhost:3196" # The gRPC server address of the Edge Node simulator (if/when needed) - e.g., localhost:3196
CA_PATH="" # The file path of the CA certificate of the target orchestrator cluster
ONBUSER="" # The orch keycloak user - to retrieve token for Infrastructure Manager SBI interactions of ENSIM
ONBPASS="" # The orch keycloak user password - to retrieve token for Infrastructure Manager SBI interactions of ENSIM
APIUSER="" # The orch keycloak user - to retrieve token for Infrastructure Manager REST API interactions - if not specified goes to default
APIPASS="" # The orch keycloak user password - to retrieve token for Infrastructure Manager REST API interactions - if not specified goes to default
PROJECT="" # The project name in which the ONBUSER and APIUSER belong to.
```

## Edge Node Simulator Deployment

Deploy the edge node simulator in the same namespace as orch-infra (Edge Infrastructure Manager).

```bash
helm upgrade --install -n orch-infra ensim \
    oci://registry-rs.edgeorchestration.intel.com/edge-orch/infra/charts/ensim \
    --set global.registry.name=registry-rs.edgeorchestration.intel.com/edge-orch/ \
    --set configArgs.server.orchFQDN=kind.internal \
    --set tlsSecretName=gateway-ca-cert

sleep 5
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=ensim -n orch-infra --timeout=5m
```

## Run Integration Tests

Set port-forward to the following targets:

```bash
kubectl port-forward svc/ensim -n orch-infra --address 0.0.0.0 3196:3196 &
kubectl port-forward svc/api -n orch-infra --address 0.0.0.0 8080:8080 &
```

### Runs day0 integration tests

```bash
ginkgo -v -r --fail-fast --race --json-report infra-tests-day0.json --output-dir . --label-filter="infra-tests-day0" ./test/infra -- \
    -project=${PROJECT} -caFilepath=${CA_PATH} -simAddress=${ENSIM_ADDR} \
    -clusterFQDN=${ORCH_FQDN} \
    -edgeAPIUser=${APIUSER}  -edgeAPIPass=${APIPASS} \
    -edgeOnboardUser=${ONBUSER} -edgeOnboardPass=${ONBPASS}
```

### Runs day1 integration tests

```bash
ginkgo -v -r --fail-fast --race --json-report infra-tests-day1.json --output-dir . --label-filter="infra-tests-day1" ./test/infra -- \
    -project=${PROJECT} -caFilepath=${CA_PATH} -simAddress=${ENSIM_ADDR} \
    -clusterFQDN=${ORCH_FQDN} \
    -edgeAPIUser=${APIUSER}  -edgeAPIPass=${APIPASS} \
    -edgeOnboardUser=${ONBUSER} -edgeOnboardPass=${ONBPASS}
```

### Runs day2 integration tests

```bash
ginkgo -v -r --fail-fast --race --json-report infra-tests-day2.json --output-dir . --label-filter="infra-tests-day2" ./test/infra --  \
    -project=${PROJECT} -caFilepath=${CA_PATH} -simAddress=${ENSIM_ADDR} \
    -clusterFQDN=${ORCH_FQDN} \
    -edgeAPIUser=${APIUSER}  -edgeAPIPass=${APIPASS} \
    -edgeOnboardUser=${ONBUSER} -edgeOnboardPass=${ONBPASS}
```

## Run hosts/locations cleanup

```bash
ginkgo -v -r --fail-fast --race --label-filter="cleanup" ./test/infra --  \
    -project=${PROJECT} -caFilepath=${CA_PATH} -simAddress=${ENSIM_ADDR} \
    -clusterFQDN=${ORCH_FQDN} \
    -edgeAPIUser=${APIUSER}  -edgeAPIPass=${APIPASS} \
    -edgeOnboardUser=${ONBUSER} -edgeOnboardPass=${ONBPASS}
```

## Kill port-forward to ensim/api

```bash
kill $(ps -eaf | grep 'kubectl' | grep 'port-forward svc/ensim' | awk '{print $2}')
kill $(ps -eaf | grep 'kubectl' | grep 'port-forward svc/api' | awk '{print $2}')
```
