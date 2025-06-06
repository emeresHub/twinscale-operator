#!/usr/bin/env bash
set -euo pipefail

# ────────────────────────────────────────────────────────────────
# 1. Configuration for Both Databases
# ────────────────────────────────────────────────────────────────
NAMESPACE="postgresql"
HELM_REPO="bitnami"
CHART="bitnami/postgresql"
VERSION="16.1.0"

# PostgreSQL 1: EventStore
ES_RELEASE="eventstore-postgresql"
ES_USER="event_user"
ES_DB="event_db"
ES_PASSWORD="eventstore-pass"

# PostgreSQL 2: HistoricalDataStore
HDS_RELEASE="historicalstore-postgresql"
HDS_USER="hds_user"
HDS_DB="hds_db"
HDS_PASSWORD="historical-pass"

# ────────────────────────────────────────────────────────────────
# 2. Ensure Namespace Exists
# ────────────────────────────────────────────────────────────────
kubectl get ns "$NAMESPACE" &>/dev/null || kubectl create namespace "$NAMESPACE"

# ────────────────────────────────────────────────────────────────
# 3. Add Bitnami Repo (if not already added)
# ────────────────────────────────────────────────────────────────
helm repo add "$HELM_REPO" https://charts.bitnami.com/bitnami || true
helm repo update

# ────────────────────────────────────────────────────────────────
# 4. Deploy PostgreSQL for EventStore
# ────────────────────────────────────────────────────────────────
helm upgrade --install "$ES_RELEASE" "$CHART" \
  --version "$VERSION" \
  --namespace "$NAMESPACE" \
  --set global.postgresql.auth.postgresPassword="$ES_PASSWORD" \
  --set global.postgresql.auth.username="$ES_USER" \
  --set global.postgresql.auth.database="$ES_DB" \
  --set primary.persistence.enabled=false

# ────────────────────────────────────────────────────────────────
# 5. Deploy PostgreSQL for HistoricalDataStore
# ────────────────────────────────────────────────────────────────
helm upgrade --install "$HDS_RELEASE" "$CHART" \
  --version "$VERSION" \
  --namespace "$NAMESPACE" \
  --set global.postgresql.auth.postgresPassword="$HDS_PASSWORD" \
  --set global.postgresql.auth.username="$HDS_USER" \
  --set global.postgresql.auth.database="$HDS_DB" \
  --set primary.persistence.enabled=false

# ────────────────────────────────────────────────────────────────
# 6. Wait for Pods to Become Ready
# ────────────────────────────────────────────────────────────────
kubectl wait --for=condition=Ready pod \
  -l app.kubernetes.io/instance="$ES_RELEASE" \
  --timeout=180s -n "$NAMESPACE"

kubectl wait --for=condition=Ready pod \
  -l app.kubernetes.io/instance="$HDS_RELEASE" \
  --timeout=180s -n "$NAMESPACE"

# ────────────────────────────────────────────────────────────────
# 7. Create Database Secrets for Application Access
# ────────────────────────────────────────────────────────────────

# Namespace where EventStore and HistoricalDataStore services live
APP_NAMESPACE="twinscale"

# Secret for EventStore
kubectl create secret generic "${ES_RELEASE}-credentials" \
  --namespace "$APP_NAMESPACE" \
  --from-literal=host="${ES_RELEASE}.${NAMESPACE}.svc.cluster.local" \
  --from-literal=database="$ES_DB" \
  --from-literal=username="$ES_USER" \
  --from-literal=password="$ES_PASSWORD" \
  --dry-run=client -o yaml | kubectl apply -f -


# Secret for HistoricalDataStore
kubectl create secret generic "${HDS_RELEASE}-credentials" \
  --namespace "$APP_NAMESPACE" \
  --from-literal=host="${HDS_RELEASE}.${NAMESPACE}.svc.cluster.local" \
  --from-literal=database="$HDS_DB" \
  --from-literal=username="$HDS_USER" \
  --from-literal=password="$HDS_PASSWORD" \
  --dry-run=client -o yaml | kubectl apply -f -


# ────────────────────────────────────────────────────────────────
# 8. Done
# ────────────────────────────────────────────────────────────────
echo "✔ Deployed two PostgreSQL databases in namespace '$NAMESPACE'"
echo "✔ Created access secrets in namespace '$APP_NAMESPACE'"
