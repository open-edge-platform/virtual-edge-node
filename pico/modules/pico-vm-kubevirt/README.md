<!--
SPDX-FileCopyrightText: 2025 Intel Corporation

SPDX-License-Identifier: Apache-2.0
-->

# VEN VM on KubeVirt

## Development Environment

### Setup

Follow these steps to set up a test environment for running a VM on KubeVirt using k3s.

1. Install k3s:

```shell
curl -sfL https://get.k3s.io | sh -
```

1. Setup KubeConfig

```shell
mkdir -p $HOME/.kube
sudo cp /etc/rancher/k3s/k3s.yaml $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config
export KUBECONFIG=$HOME/.kube/config
```

1. Verify K3s is running:

```shell
kubectl get nodes
kubectl get pods -A
```

1. Install KubeVirt:

```shell
export VERSION=$(curl -s https://storage.googleapis.com/kubevirt-prow/release/kubevirt/kubevirt/stable.txt)
echo $VERSION
kubectl create -f https://github.com/kubevirt/kubevirt/releases/download/${VERSION}/kubevirt-operator.yaml
kubectl create -f https://github.com/kubevirt/kubevirt/releases/download/${VERSION}/kubevirt-cr.yaml
```

1. Verify KubeVirt is running:

```shell
kubectl get all -n kubevirt
```

1. Install CDI

```shell
export TAG=$(curl -s -w %{redirect_url} https://github.com/kubevirt/containerized-data-importer/releases/latest)
export VERSION=$(echo ${TAG##*/})
kubectl create -f https://github.com/kubevirt/containerized-data-importer/releases/download/$VERSION/cdi-operator.yaml
kubectl create -f https://github.com/kubevirt/containerized-data-importer/releases/download/$VERSION/cdi-cr.yaml
```

1. Expose CDI

```shell
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: cdi-uploadproxy-nodeport
  namespace: cdi
  labels:
    cdi.kubevirt.io: "cdi-uploadproxy"
spec:
  type: NodePort
  ports:
    - port: 443
      targetPort: 8443
      nodePort: 31001
      protocol: TCP
  selector:
    cdi.kubevirt.io: cdi-uploadproxy
EOF
```

1. Install virtctl:

```shell
VERSION=$(kubectl get kubevirt.kubevirt.io/kubevirt -n kubevirt -o=jsonpath="{.status.observedKubeVirtVersion}")
ARCH=$(uname -s | tr A-Z a-z)-$(uname -m | sed 's/x86_64/amd64/')
curl -L -o virtctl https://github.com/kubevirt/kubevirt/releases/download/${VERSION}/virtctl-${VERSION}-${ARCH}
chmod +x virtctl
sudo install virtctl /usr/local/bin
```

1. Verify virtctl is installed:

```shell
virtctl version
```

1. Add cdi-uploadproxy to /etc/hosts:

```shell
nano /etc/hosts

# Add the following line
127.0.0.1 cdi-uploadproxy
```

1. Deploy test VM

```shell
kubectl create namespace vm
kubectl apply -f https://kubevirt.io/labs/manifests/vm.yaml -n vm
kubectl patch virtualmachine testvm --type merge -p '{"spec":{"runStrategy":"Always"}}' -n vm
kubectl get vms -n vm
```

### Clean up

1. Clean up test VM

```shell
kubectl delete virtualmachine testvm -n vm
kubectl delete namespace vm
```

1. Clean up KubeVirt and CDI

```shell
export TAG=$(curl -s -w %{redirect_url} https://github.com/kubevirt/containerized-data-importer/releases/latest)
export VERSION=$(echo ${TAG##*/})
kubectl delete -f https://github.com/kubevirt/containerized-data-importer/releases/download/$VERSION/cdi-operator.yaml
kubectl delete -f https://github.com/kubevirt/containerized-data-importer/releases/download/$VERSION/cdi-cr.yaml

export VERSION=$(curl -s https://storage.googleapis.com/kubevirt-prow/release/kubevirt/kubevirt/stable.txt)
echo $VERSION
kubectl delete -f https://github.com/kubevirt/kubevirt/releases/download/${VERSION}/kubevirt-operator.yaml
kubectl delete -f https://github.com/kubevirt/kubevirt/releases/download/${VERSION}/kubevirt-cr.yaml
```

1. Uninstall k3s:

```shell
/usr/local/bin/k3s-uninstall.sh
```
