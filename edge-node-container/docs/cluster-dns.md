# How to use cluster-dns

1.Configure the env file and copy to /etc/cluster-dns/env. <br>
  a. CLUSTER_IP is the ip address of Orchestrator. <br>
  b. CLUSTER_FQDN is the suffix of the domain name of the Orchestrator. <br>
2.Copy the cluster-dns.sh to /usr/bin/code-cluster-dns.sh <br>
3.Copy the cluster-dns.service to /etc/systemd/system <br>
4.systemctl enable cluster-dns <br>
5.systemctl start cluster-dns  <br>

## The functionality of cluster-dns

The functionality of the cluster-dns is to patch the host alias
field to the deployment of cattle, since you can't connect to the prviate
Orchestrator without any hosts setting.

If the Orchestrator is public and registered on the dns server, you not need
to use this script.
