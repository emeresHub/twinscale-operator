#!/usr/bin/env bash
set -euo pipefail

# ──────────────────────────────────────────────────────────────────────────────
# 1. Configuration
# ──────────────────────────────────────────────────────────────────────────────
NAMESPACE="postgresql"
RELEASE="twinscale-postgresql"
HELM_REPO="bitnami"
CHART="bitnami/postgresql"
VERSION="16.1.0"              # pin to a known chart version
PG_PASSWORD="postgresql"      # change in prod!

# ──────────────────────────────────────────────────────────────────────────────
# 2. Create namespace if needed
# ──────────────────────────────────────────────────────────────────────────────
kubectl get ns "$NAMESPACE" &>/dev/null \
  || kubectl create namespace "$NAMESPACE"

# ──────────────────────────────────────────────────────────────────────────────
# 3. Add & update Helm repo
# ──────────────────────────────────────────────────────────────────────────────
helm repo add "$HELM_REPO" https://charts.bitnami.com/bitnami
helm repo update

# ──────────────────────────────────────────────────────────────────────────────
# 4. Install or upgrade PostgreSQL
# ──────────────────────────────────────────────────────────────────────────────
helm upgrade --install "$RELEASE" "$CHART" \
  --version "$VERSION" \
  --namespace "$NAMESPACE" \
  --set global.postgresql.auth.postgresPassword="$PG_PASSWORD" \
  --set global.postgresql.auth.username=twinscale_user \
  --set global.postgresql.auth.database=twinscale_db \
  --set primary.persistence.enabled=false

# ──────────────────────────────────────────────────────────────────────────────
# 5. Wait for the primary pod to be Ready (avoids the StatefulSet name confusion)
# ──────────────────────────────────────────────────────────────────────────────
kubectl wait --for=condition=Ready pod \
  -l app.kubernetes.io/instance="$RELEASE" \
  --timeout=180s -n "$NAMESPACE"

# ──────────────────────────────────────────────────────────────────────────────
# 6. (Optional) Create a credentials Secret for downstream consumers
# ──────────────────────────────────────────────────────────────────────────────
kubectl create secret generic "${RELEASE}-credentials" \
  --namespace "$NAMESPACE" \
  --from-literal=username=twinscale_user \
  --from-literal=password="$PG_PASSWORD" \
  --from-literal=database=twinscale_db \
  --dry-run=client -o yaml | kubectl apply -f -

echo "✔ PostgreSQL '$RELEASE' is up in namespace '$NAMESPACE'"
