# Security Policy
Intel is committed to rapidly addressing security vulnerabilities affecting our customers and providing clear guidance on the solution, impact, severity and mitigation. 

## Reporting a Vulnerability
Please report any security vulnerabilities in this project utilizing the guidelines [here](https://www.intel.com/content/www/us/en/security-center/vulnerability-handling-guidelines.html).

## About [Virtual Edge Node Provisioning (VEN)](./vm-provisioning)

- VEN is not part of EMF (Edge Manageability Framework).
- VEN is developed and released for testing purposes only.
- VEN is used by the Continuous Integration pipeline of EMF to
  run integration tests in the scope of onboarding and provisioning
  of Edge Nodes.
- Developers can use the Virtual Edge Node to create and manage VMs 
  that mirror production environments without need for physical hardware.
- VEN is useful for developers and testers who need to simulate and test 
  the onboarding and provisioning processes of virtual environments using 
  development code.
- It provides set of scripts, templates, and configurations to deploy VENs
  on an Edge Orchstrator.

## About [Edge Node in a Container (ENiC)](./edge-node-container)

- ENiC is not part of EMF (Edge Manageability Framework);
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
