#!/usr/bin/env bash

SCRIPT_PATH=$(dirname "$0")

# RabbitMQ
RABBITMQ_VERSION=v2.4.0
RABBITMQ_CERT_MANAGER_VERSION=v1.11.1
RABBITMQ_MESSAGING_TOPOLOGY_OPERATOR_VERSION=v1.10.3
KNATIVE_RABBITMQ_BROKER_VERSION=v1.9.1

### Execute Installation scripts

# RabbitMQ Cluster
kubectl apply -f https://github.com/rabbitmq/cluster-operator/releases/download/${RABBITMQ_VERSION}/cluster-operator.yml
kubectl apply -f ${SCRIPT_PATH}/brokers/cluster-operator/1-cluster-operator.yml
kubectl wait --for=condition=available --timeout=200s --all deployments --namespace rabbitmq-system

kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/${RABBITMQ_CERT_MANAGER_VERSION}/cert-manager.yaml
kubectl apply -f ${SCRIPT_PATH}/brokers/cluster-operator/2-cert-manager.yaml
kubectl wait --for=condition=available --timeout=200s --all deployments --namespace cert-manager

kubectl apply -f https://github.com/rabbitmq/messaging-topology-operator/releases/download/${RABBITMQ_MESSAGING_TOPOLOGY_OPERATOR_VERSION}/messaging-topology-operator-with-certmanager.yaml
kubectl apply -f ${SCRIPT_PATH}/brokers/cluster-operator/3-messaging-topology-operator-with-certmanager.yaml
kubectl wait --for=condition=available --timeout=200s --all deployments --namespace rabbitmq-system

# RabbitMQ Cluster
kubectl apply -f ${SCRIPT_PATH}/brokers/rabbitmq-cluster -n twinscale
kubectl wait --for=condition=available --timeout=200s --all deployments --namespace knative-eventing
kubectl wait --for=condition=available --timeout=200s --all deployments --namespace twinscale
kubectl wait --for=condition=Ready --timeout=200s --all pods --namespace twinscale

# RabbitMQ Eventing
kubectl apply -f https://github.com/knative-sandbox/eventing-rabbitmq/releases/download/knative-${KNATIVE_RABBITMQ_BROKER_VERSION}/rabbitmq-broker.yaml
kubectl apply -f ${SCRIPT_PATH}/brokers/rabbitmq-broker/1-rabbitmq-broker.yaml
kubectl wait --for=condition=available --timeout=200s --all deployments --namespace knative-eventing

# RabbitMQ Broker
kubectl apply -f ${SCRIPT_PATH}/brokers/rabbitmq-broker/1-rabbitmq-broker.yaml -n knative-eventing
kubectl apply -f ${SCRIPT_PATH}/brokers/rabbitmq-broker/2-rabbitmq-config.yaml -n twinscale
kubectl apply -f ${SCRIPT_PATH}/brokers/rabbitmq-broker/3-rabbitmq-broker.yaml -n twinscale
kubectl wait --for=condition=available --timeout=200s --all deployments --namespace knative-eventing
kubectl wait --for=condition=available --timeout=200s --all deployments --namespace twinscale
kubectl wait --for=condition=Ready --timeout=200s --all pods --namespace twinscale

echo "Setup broker script has finished"

# Uninstall
# kubectl delete -f ${SCRIPT_PATH}/brokers/rabbitmq-broker -n twinscale
# kubectl delete -f ${SCRIPT_PATH}/brokers/rabbitmq-broker/1-rabbitmq-broker.yaml
# kubectl delete -f ${SCRIPT_PATH}/brokers/rabbitmq-cluster -n twinscale
# kubectl delete -f ${SCRIPT_PATH}/brokers/rabbitmq-cluster -n knative-eventing
# kubectl delete -f ${SCRIPT_PATH}/brokers/cluster-operator/3-messaging-topology-operator-with-certmanager.yaml
# kubectl delete -f ${SCRIPT_PATH}/brokers/cluster-operator/2-cert-manager.yaml
# kubectl delete -f ${SCRIPT_PATH}/brokers/cluster-operator/1-cluster-operator.yml
