#!/bin/bash

set -e

echo "Creating namespace if not exists..."
kubectl create namespace twinscale --dry-run=client -o yaml | kubectl apply -f -

echo "Applying Prometheus configuration..."
kubectl apply -f ./hack/monitoring/prometheus-config.yaml

echo "Deploying Prometheus..."
kubectl apply -f ./hack/monitoring/prometheus.yaml

echo "Deploying Grafana..."
kubectl apply -f ./hack/monitoring/grafana.yaml

echo "Installing Redis using Helm..."
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update
helm upgrade --install redis bitnami/redis \
  -n twinscale \
  -f ./hack/monitoring/redis-values.yaml

echo "Monitoring stack deployed in namespace 'twinscale'."
