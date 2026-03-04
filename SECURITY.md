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

