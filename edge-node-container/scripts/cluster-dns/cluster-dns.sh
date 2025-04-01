#!/usr/bin/env bash

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# Script Name: cluster-dns.sh
# Description: This script is meant to run as systemd service
# and is used to path the hostAliases of the agents requiring
# DNS resolution of the ORCH_FQDN endpoints

function patch-orchestrator-service-proxy-agent () {
  # shellcheck disable=SC2154
  PATCH_JSON=$(cat << EOF
{
  "spec": {
    "template": {
      "spec": {
        "hostAliases": [
          {
            "hostnames": ["app-orch.${CLUSTER_FQDN}"],
            "ip": "${CLUSTER_IP}"
          }
        ]
      }
    }
  }
}
EOF
)
  REQ_JSON=$(echo "${PATCH_JSON}"| jq -c)
  echo "Patch orch-service-proxy-proxy-agent"
  orchSrvProxy=$(kubectl get deployment -n orchestrator-system orchestrator-service-proxy-proxy-agent -o json | jq ".spec.template.spec.hostAliases")
  while [ -z "${orchSrvProxy}" ]; do
    echo "Wait for orchestrator-service-proxy-agent"
    orchSrvProxy=$(kubectl get deployment -n orchestrator-system orchestrator-service-proxy-proxy-agent -o json | jq ".spec.template.spec.hostAliases")
    sleep 5
  done

  orchSrvProxy=$(kubectl get deployment -n orchestrator-system orchestrator-service-proxy-proxy-agent -o json | jq ".spec.template.spec.hostAliases" | grep "${CLUSTER_IP}")
  while [ -z "${orchSrvProxy}" ]; do
    echo -e "Adding \n${REQ_JSON}\n to orchestrator-service-proxy-agent hostAliases"
    kubectl -n orchestrator-system patch deployment orchestrator-service-proxy-proxy-agent --patch "${REQ_JSON}"
    sleep 5
    orchSrvProxy=$(kubectl get deployment -n orchestrator-system orchestrator-service-proxy-proxy-agent -o json | jq ".spec.template.spec.hostAliases" | grep "${CLUSTER_IP}")
  done

}

function patch-fleet-agent () {
  PATCH_JSON=$(cat << EOF
{
  "spec": {
    "template": {
      "spec": {
        "hostAliases": [
          {
            "hostnames": ["rancher.${CLUSTER_FQDN}"],
            "ip": "${CLUSTER_IP}"
          }
        ]
      }
    }
  }
}
EOF
)
        REQ_JSON=$(echo "${PATCH_JSON}"| jq -c)
        echo "Patch fleet-agent"
        while true; do
        echo "Waiting for fleet-agent-0 to be running..."
        STATUS=$(kubectl get pod fleet-agent-0 -n cattle-fleet-system -o=jsonpath='{.status.phase}')
        if [ "${STATUS}" == "Running" ]; then
                #kubectl -n cattle-fleet-system patch statefulsets fleet-agent --patch "${REQ_JSON}"
                echo "fleet-agent is Running."
                break
        else
                kubectl -n cattle-fleet-system patch statefulset fleet-agent --patch "${REQ_JSON}"
                sleep 5
                kubectl get statefulset -n cattle-fleet-system fleet-agent -o json | jq ".spec.template.spec.hostAliases"
                kubectl delete pod fleet-agent-0 -n cattle-fleet-system
        fi
        sleep 30
        done
}

function patch-cattle-cluster-agent() {
  PATCH_JSON=$(cat << EOF
{
  "spec": {
    "template": {
      "spec": {
        "hostAliases": [
          {
            "hostnames": ["rancher.${CLUSTER_FQDN}"],
            "ip": "${CLUSTER_IP}"
          }
        ]
      }
    }
  }
}
EOF
)
  REQ_JSON=$(echo "${PATCH_JSON}"| jq -c)
  echo "Patch cattle-cluster-agent"
  cattleDns=$(kubectl get deployment -n cattle-system cattle-cluster-agent -o json | jq ".spec.template.spec.hostAliases")
  while [ -z "$cattleDns" ]; do
    echo "Wait for cattle-cluster-agent startup"
    cattleDns=$(kubectl get deployment -n cattle-system cattle-cluster-agent -o json| jq ".spec.template.spec.hostAliases")
    sleep 5
  done

  while [[ "$cattleDns" ==  "null" ]]; do
    echo -e "Adding \n${REQ_JSON}\n to cattle-cluster-agent hostAliases"
    kubectl patch deployment -n cattle-system cattle-cluster-agent --patch "${REQ_JSON}"
    sleep 5
    cattleDns=$(kubectl get deployment -n cattle-system cattle-cluster-agent -o json | jq ".spec.template.spec.hostAliases")
  done
}

function patch-hostAliases() {
  export KUBECONFIG=/etc/rancher/rke2/rke2.yaml
  export PATH=${PATH}:/var/lib/rancher/rke2/bin
  # Never ending service - to support multiple installations
  while true; do
    # if kubectl is not present yet - means that the
    # cluster installation is not yet started
    until [ -f /var/lib/rancher/rke2/bin/kubectl ]
    do
      echo "Wait for cluster setup"
      sleep 5
    done

    # Path one-by-one all the agents
    patch-cattle-cluster-agent
    patch-fleet-agent
    patch-orchestrator-service-proxy-agent

    # if kubectl is gone - means that the cluster
    # was removed
    until [ ! -f /var/lib/rancher/rke2/bin/kubectl ]
    do
      echo "Wait for cluster teardown"
      sleep 5
    done

  done
}

function runAsService() {
        patch-hostAliases
}


main() {
        subcommand="$1"
        shift

        case "$subcommand" in
        service)
                runAsService "$@"
                ;;
        *)
                echo "unknown subcommand"
                return 1
                ;;
        esac
}

main "$@"
