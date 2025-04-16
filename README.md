# Virtual Edge Node

## Overview

The Virtual Edge Node (VEN) is designed to streamline the onboarding and provisioning of virtual machines, as well as
the deployment, management, and testing of edge computing applications. It offers a virtualized platform that replicates
the functionality of physical edge devices, enabling developers and testers to simulate real-world scenarios without
requiring actual hardware.

**Note: This repository is intended for Edge Developers testing environments and is not meant for production
usecase or deployment on live systems.**

The repository supports Day 0 provisioning of the Virtual Edge Nodes for the Edge Manageability Framework and includes:

- [**VM-Provisioning**](vm-provisioning/): provides a set of scripts, templates, and configurations designed to streamline
  and automate the initial setup and deployment of virtual machines (VMs) during the Day 0 provisioning phase on an Edge
  Orchestrator. It utilizes Vagrant and libvirt APIs to ensure efficient and smooth VM provisioning.
- [**Edge Node in a Container**](edge-node-container/): contains an emulated version of an edge node in a container,
  for testing purposes only.
- [**Edge Node Simulator**](edge-node-simulator/): contains a simulator for edge nodes with the Infrastructure Manager,
  for testing purposes only.

Read more about Virtual Edge Node in the [User Guide][user-guide-url].

Navigate through the folders to get started, develop, and contribute to Virtual Edge Node.

[user-guide-url]: https://docs.openedgeplatform.intel.com/edge-manage-docs/main/developer_guide/virtual_edge_node/index.html

## Contribute

To learn how to contribute to the project, see the [Contributor's
Guide](https://docs.openedgeplatform.intel.com/edge-manage-docs/main/developer_guide/contributor_guide/index.html).

## Community and Support

To learn more about the project, its community, and governance, visit
the [Edge Orchestrator Community](https://docs.openedgeplatform.intel.com/edge-manage-docs/main/index.html).

For support, start with [Troubleshooting](https://docs.openedgeplatform.intel.com/edge-manage-docs/main/developer_guide/troubleshooting/index.html)

## License

Each component of the Virtual Edge Node is licensed under [Apache 2.0][apache-license].

Last Updated Date: April 15, 2025

[apache-license]: https://www.apache.org/licenses/LICENSE-2.0
