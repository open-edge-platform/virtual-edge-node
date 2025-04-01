# Architecture

The ensim is written using the ensim source code, they are in different folders.

## Simulated Edge Node

The simulated edge node is meant to exercise Edge Infrastructure Manager
southbound interfaces for the main procedures of: onboarding, provisioning, agents execution.
When a simulated edge node is initiated it goes through
each one of these procedures (explained below).
If any of these processes fail, then the simulated edge node fails.
See below the C4 component diagram of the simulated edge node.

```mermaid
C4Component
    title Component diagram ENSIM

    Container_Boundary(ensim, "ENSIM") {
        Container_Boundary(onbprov, "Onboard/Provision") {
            Component(onboard, "Interactive Onboarding", "Interactive", "All interactive onboarding mechanisms.")
            
            Component(nio, "Non-Interactive Onboarding", "NIO", "All non-interactive onboarding mechanisms.")
        
            Component(prov, "Provisioning", "Tinkerbell worker", "Executes tinkerbell workflow actions.")
        }

        Container_Boundary(token, "Token Manager") {
            Component(token-manager, "Token Manager", "", "Periodic Keycloak token refresh")       
        }

        Container_Boundary(agents, "Agents (Simulated)") {
            Component(update-agent, "Update Agent", "Update Agent", "Updates maintenance status")
            Component(node-agent, "Node Agent", "Node Agent", "Updates host/instance status")
            Component(hd-agent, "HW Discovery Agent", "HDA", "Updates HW info")
            Component(telemetry-agent, "Telemetry Agent", "Telemetry Agent", "Set telemetry config")
        }
    }
    Container_Boundary(orch, "Orchestrator") {
        Container_Boundary(infra, "Edge Infrastructure Manager") {
            System_Ext(om, "Onboarding Manager", "Onboarding Manager")
            System_Ext(tinker, "Tinkerbell server", "Tinkerbell Workflows")
            System_Ext(mrm, "Maintenance Resource Manager", "MRM")
            System_Ext(hrm, "Host Resource Manager", "HRM")
            System_Ext(trm, "Telemetry Resource Manager", "TRM")
        }
        System_Ext(kc, "Keycloak", "Keycloak Token Manager")
    }
    Rel(onboard, om, "CreateNodes", "gRPC")
    Rel(nio, om, "OnboardNodeStream", "gRPC")
    
    Rel(prov, tinker, "GetWorkflowContexts/ReportActionStatus", "gRPC")
    
    Rel(token-manager, kc, "GET /token", "JSON/HTTPS")
    
    Rel(node-agent, hrm, "UpdateInstanceStateStatusByHostGUID", "gRPC")
    Rel(hd-agent, hrm, "UpdateHostSystemInfoByGUID", "gRPC")
    Rel(update-agent, mrm, "PlatformUpdateStatus", "gRPC")
    Rel(telemetry-agent, trm, "GetTelemetryConfigByGUID", "gRPC")
  ```

The teardown procedure is a hack of simulated edge node, that facilitates its decomission in
Edge Infrastructure Manager (removal of its artifacts/resources).
While an actual edge node, if decomissioned from Edge Infrastructure Manager,
would need to have its artifacts/resources manually removed by the
Edge Infrastructure Manager operator/manager.

A requirement for the ensim is to create the folder where its credentials
are going to be stored, consisting of:
client_id, client_secret, client_token, agents tokens.

As Edge Infrastructure Manager supports multi-tenancy,
it is important to provide the correct users/passwords for the simulated edge node.
The credentials provided for `onbUser / onbPasswd` are set to perform the main procedures.
While `apiUser / apiPasswd` are meant to be used only or the teardown procedure.
And `projectID` is meant to be used to identify which tenant the edge node
is associated with in Edge Infrastructure Manager.
All the creation of users/passwords and projects can be done in the orchestrator,
and are not in the scope this document.

### Onboarding

The onboarding in Edge Infrastructure Manager can be executed via
2 southbound interfaces (of Onboarding Manager).
One of them referenced as interactive onboarding,
in simulated edge node params defined as `enableSouthOnboard: true`,
while the other referenced as non-interactive onboarding,
in simulated edge node params defined as `enableNIO: false`.
Only one of them must be specified as `true` for the onboarding process.
In Edge Infrastructure Manager, after this process the edge node appears
with the onboarding status as `onboarded`.

For `south onboard` option the ensim depends on the configuration of the
edge node via web-ui or REST API, or that the auto provision option of
Edge Infrastructure Manager provider is enable for its project.

For the `nio` option the ensim depends on the registration of the simulated edge node via
the web-ui or REST API, with matching uuid and/or serial number.

Examples of these procedures are shown in the integration tests of simulated edge node.
Details of onboarding/provisioning of edge nodes can be seen in
Edge Infrastructure Manager documentation.

### Provisioning

The provision process is simulated by the execution of Tinkerbell workflow actions.
An workflow is retrieved by the simulated edge node and all its actions are confirmed with success.
Only 2 of those actions are actually executed by the simulated edge node,
the writing of `client_id` and `client_secret` into files.
In Edge Infrastructure Manager, after this process the edge node appears
with the provisioning status as `provisioned`.

### Keycloak Token

The client `id` and `secret` credentials retrieved by tinker are used by
edge node to retrieve the keycloack token, so the agents can be properly initialized.
This is the same procedure as its done by an actual edge node.

### Token Manager

The token manager is a process running in background that refreshes the
keycloak client token every 50 minutes. As the token expires every 1 hour.
This allows the tokens of all the agents to be refreshed by the node-agent
so they maintain their communication with Edge Infrastructure Manager.

### Agents (Simulated) Execution

All agents (i.e., node, update, hardware-discovery, telemetry) have their calls simulated.
Each one of them have fake information provided in the calls to Edge Infrastructure Manager.
For instance, hardware discovery agent feeds Edge Infrastructure Manager
with fake info of storages, USBs, GPUs, NICs.
Each agent is set to run in the same period as in an actual edge node, so similar workload is executed.

### Teardown

As mentioned, when enabled, the teardown performs the communication of the edge node
to the Edge Infrastructure Manager REST API to cleanup its registries automatically.
This is just a hack to facilitate the interaction of simulated edge nodes with
Edge Infrastructure Manager (i.e., fast prototyping, testing, feature changes).

See below the flowchart diagram of the simulated edge node procedures.

```mermaid
flowchart TD;
    A(Init) --> |Started|B{Onboard};
    B --> C[Interactive Onboarding];
    B --> D[Non-Interactive Onboarding];
    C --> |Onboarding Done|E[Provisioning];
    D --> |Onboarding Done|E[Provisioning];
    E --> |Provisioning Done|F[Client Token];
    F --> |Got Client Token|G[Token Manager];
    G --> |Got Agents Tokens|H[Agents];
    H --> |Running|I(Execution Events);
    I --> |Stopped|J(Teardown);
```

## Edge Node Simulator (ensim)

Different from simulated edge node, the simulator allows the execution of
multiple simulated edge nodes in parallel via the same golang process.
It uses the same code base from a single simulated edge node, plus a client/server interface
to easily interact with simulated edge nodes (create/delete/list/get/update).

### Server

The ensim server has a store to maintain instances of simulated edge nodes,
and provides a gRPC northbound interface to interact with such store.
In the server, each edge node is identified by its UUID, the same one used by Edge Infrastructure Manager.

Among the ensim server operations, it allows the following operations:

- Create Node: instantiates a single ensim, with given credentials (users/passwords and projectID);
- Delete Node: given its UUID, it stops all ensim operations and performs its teardown (when enabled);
- Update Node: it allows turning ON/OFF each one of the ensim agents;
- Create/Delete in batches: allows to create/delete many ensim instances.

The ensim server is stateless, if restarted it won't recover all the previous ensim running on it.
However it has as a parameter a base folder, where it uses to store all ensim credentials,
indexed by UUIDs as folders.
Upon create/delete of ensim the ensim server creates/deletes the respective UUID folder of the ensim.

### Client

The client is a CLI implementation, simply to facilitate the interactions with the server.
With its input parameters it provides helper ways to provide arguments (e.g., users/projectID) to the ensim.
The client CLI does not provide the option to update an agent in the ensim yet.

The CLI allows the options to:

- `Create Node`: creates a single edge node.
- `List Node`: lists all edge nodes in ensim server.
- `Get Node`: gets the information of a specific edge node given its UUID.
- `Delete Node`: deletes a specific edge node given its UUID.
- `Create Nodes`: creates multiple edge nodes in ensim server in parallel using batches.
- `Delete Nodes`: deletes a specified amount of edge nodes from the ensim server.
Specify 0 (zero) to delete all of them.

Each edge node in ensim server contains the following information (see an example below):

- uuid: the edge node UUID;
- credentials: contains the usernames and passwords to access the orchestrator cluster,
and the edge node project ID;
- status: set of information about the agents status and all the status
of the edge node procedures (e.g., credentials, teardown, onboarding);
- agentsStates: info about the state of agents running in the edge node.
It is possible to turn agents ON/OFF via the ensim client (not via CLI).

```json
{
  "uuid":  "7f14361b-f0f3-4695-91f0-5bc20af0b9c5",
  "credentials":  {
    "projectId":  "8e202529-f980-4bc2-9e34-e2e927336d36",
    "onboardUsername":  "adminTestUser1",
    "onboardPassword":  "adminTestPassword1!",
    "apiUsername":  "adminTestUser1",
    "apiPassword":  "adminTestPassword1!"
  },
  "status":  [
    {
      "source":  "STATUS_SOURCE_UPDATE_AGENT",
      "mode":  "STATUS_MODE_OK",
      "details":  "status_type:STATUS_TYPE_UP_TO_DATE"
    },
    {
      "source":  "STATUS_SOURCE_HD_AGENT",
      "mode":  "STATUS_MODE_OK"
    },
    {
      "source":  "STATUS_SOURCE_TELEMETRY_AGENT",
      "mode":  "STATUS_MODE_OK"
    },
    {
      "source":  "STATUS_SOURCE_CREDENTIALS",
      "mode":  "STATUS_MODE_OK"
    },
    {
      "source":  "STATUS_SOURCE_SETUP",
      "mode":  "STATUS_MODE_OK",
      "details":  "succefully onboarded/started"
    },
    {
      "source":  "STATUS_SOURCE_TEARDOWN",
      "mode":  "STATUS_MODE_OK"
    },
    {
      "source":  "STATUS_SOURCE_NODE_AGENT",
      "mode":  "STATUS_MODE_OK"
    },
    {
      "source":  "STATUS_SOURCE_REQUIREMENTS",
      "mode":  "STATUS_MODE_OK"
    },
    {
      "source":  "STATUS_SOURCE_ONBOARDED",
      "mode":  "STATUS_MODE_OK",
      "details":  "Onboard successful"
    },
    {
      "source":  "STATUS_SOURCE_PROVISIONED",
      "mode":  "STATUS_MODE_OK"
    }
  ],
  "agentsStates":  [
    {
      "desiredState":  "AGENT_STATE_ON",
      "currentState":  "AGENT_STATE_ON",
      "agentType":  "AGENT_TYPE_NODE"
    },
    {
      "desiredState":  "AGENT_STATE_ON",
      "currentState":  "AGENT_STATE_ON",
      "agentType":  "AGENT_TYPE_UPDATE"
    },
    {
      "desiredState":  "AGENT_STATE_ON",
      "currentState":  "AGENT_STATE_ON",
      "agentType":  "AGENT_TYPE_HD"
    },
    {
      "desiredState":  "AGENT_STATE_ON",
      "currentState":  "AGENT_STATE_ON",
      "agentType":  "AGENT_TYPE_TELEMETRY"
    }
  ]
}
```
