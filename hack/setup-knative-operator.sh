#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

SCRIPT_PATH="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

KNATIVE_VERSION="v1.10.0"
KNATIVE_OPERATOR_VERSION="v1.11.3"
ISTIOCTL_VERSION="1.18.2"

# helper to wait for specific deployments
wait_deploy() {
  local ns=$1; shift
  for deploy in "$@"; do
    echo "Waiting for deployment/${deploy} in namespace ${ns}"
    kubectl rollout status deployment/"${deploy}" -n "${ns}" --timeout=180s
  done
}

# 0. Ensure istioctl is present
if ! command -v istioctl &>/dev/null; then
  echo "istioctl not found—downloading Istio ${ISTIOCTL_VERSION}"
  curl -L https://istio.io/downloadIstio | ISTIO_VERSION="${ISTIOCTL_VERSION}" sh -
  export PATH="$PWD/istio-${ISTIOCTL_VERSION}/bin:$PATH"
fi

# 1. Install Knative Operator
kubectl apply -f "https://github.com/knative/operator/releases/download/knative-${KNATIVE_OPERATOR_VERSION}/operator.yaml"
kubectl wait --for=condition=available --timeout=200s \
  deployment/knative-operator deployment/operator-webhook

# 2. Install Istio
istioctl install --set profile=demo -y
kubectl create ns istio-system --dry-run=client -o yaml | kubectl apply -f -
wait_deploy istio-system istiod

# 3. Install Knative Serving CR
kubectl create ns knative-serving --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -f "${SCRIPT_PATH}/knative-operator/knative-serving.yaml"
# wait for the Serving CR to report Ready
kubectl wait --for=condition=Ready --timeout=300s \
  knativeserving/knative-serving -n knative-serving
# now the operator has created the core Deployments
wait_deploy knative-serving activator autoscaler controller webhook

# 4. Install net-istio plugin
kubectl apply -f "https://github.com/knative/net-istio/releases/download/knative-${KNATIVE_VERSION}/net-istio.yaml"
wait_deploy knative-serving net-istio-controller net-istio-webhook

# 5. Install Knative Eventing CR
kubectl create ns knative-eventing --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -f "${SCRIPT_PATH}/knative-operator/knative-eventing.yaml"
# wait for the Eventing CR to report Ready
kubectl wait --for=condition=Ready --timeout=300s \
  knativeeventing/knative-eventing -n knative-eventing
# then its Deployments appear
wait_deploy knative-eventing eventing-controller eventing-webhook

echo "Knative setup script has finished"
