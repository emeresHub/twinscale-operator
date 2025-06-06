#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

NAMESPACE="twinscale"
RELEASE="twinscale-graph-redis"
CHART="bitnami/redis"

REDIS_PASSWORD="eYVX7EwVmmxKPCDmwMtyKVge8oLd2t81"
MASTER_SVC="twinscale-graph-redis-master"
REPLICA_SVC="twinscale-graph-redis-replicas"

echo "Creating namespace '${NAMESPACE}' (if not present)..."
kubectl create namespace "${NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -

echo "Adding Bitnami Helm repo..."
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update

echo "Installing Redis release '${RELEASE}'..."
helm install "${RELEASE}" "${CHART}" \
  --namespace "${NAMESPACE}" \
  --set auth.password="${REDIS_PASSWORD}" \
  --set master.service.name="${MASTER_SVC}" \
  --set replica.service.name="${REPLICA_SVC}"

echo "Redis '${RELEASE}' deployed into namespace '${NAMESPACE}'."
