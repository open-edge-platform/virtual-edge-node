# Integration Tests

TODO: update, examples are outdated.

The following steps provide guidance on how to run the scale tests for days 1 and 2 using the Infrastructure Manager simulator.
All test cases have descriptions in their respective files/definitions.

## Requirements

For all the day1 and day2 tests the following environment variables need to be defined.

```bash
FQDN="" # The FQDN of the target orchestrator cluster
ENSIM_ADDR="" # The gRPC server address of the Infrastructure Manager simulator (if/when needed) - e.g., localhost:3196
CA_PATH="" # The file path of the CA certificate of the target orchestrator cluster
API_URL="" # Defines the Infrastructure Manager REST API URL of the target orchestrator cluster
KCUSER="" # The orch keycloak user - to retrieve token for Infrastructure Manager REST API interactions - if not specified goes to default
KCPASS="" # The orch keycloak user password - to retrieve token for Infrastructure Manager REST API interactions - if not specified goes to default
```

## Infrastructure Manager Simulator - Adds/Dels simulated edge nodes

Compile and run the Infrastructure Manager simulator, for instance:

```bash
make go-build
cd out/ensim/
 ./server/main -globalLogLevel debug  -orchCAPath $CA_PATH  -orchFQDN $FQDN -oamServerAddress 0.0.0.0:6379
```

Use the interactive CLI of en-sim client to add edge nodes.
Select Create Nodes option, define amount of hosts and the batch size (50 is recommended).

```bash
./client/main -addressSimulator $ENSIM_ADDR
```

## Run day 1

All test cases in day 1 use keycloak user/passwd to retrieve token to Infrastructure Manager REST API interactions.

```bash
go test -timeout=60m -count=1 -v ./test/day1/ -orchFQDN=$FQDN  -apiURL=$API_URL -caFilepath=$CA_PATH -keyCloakUser=$KCUSER -keyCloakPass=$KCPASS -run TestDay1_Case01
```

## Run day 2

All test cases in day 2 use keycloak user/passwd to retrieve token to Infrastructure Manager REST API interactions.

```bash
go test -timeout=60m -count=1 -v ./test/day2/ -orchFQDN=$FQDN -apiURL=$API_URL -caFilepath=$CA_PATH -keyCloakUser=$KCUSER -keyCloakPass=$KCPASS -run TestDay2_Case01
```
