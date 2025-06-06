package historicaldatastore

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev0 "github.com/emereshub/twinscale-operator/api/core/v0"
	dtdv0 "github.com/emereshub/twinscale-operator/api/dtd/v0"
	knative "github.com/emereshub/twinscale-operator/pkg/third-party/knative"
	"github.com/emereshub/twinscale-operator/pkg/third-party/rabbitmq"

	rabbitmqv1beta1 "github.com/rabbitmq/messaging-topology-operator/api/v1beta1"
	kEventing "knative.dev/eventing/pkg/apis/eventing/v1"
	kserving "knative.dev/serving/pkg/apis/serving/v1"
)

const (
	HISTORICAL_DATA_STORE_SERVICE = "twinscale-historical-data-store"
	BROKER_NAME                   = "twinscale"
)

type HistoricalDataStore interface {
	GetService(hds *corev0.HistoricalDataStore) *kserving.Service
	MergeService(current, desired *kserving.Service) *kserving.Service

	GetTrigger(hds *corev0.HistoricalDataStore) *kEventing.Trigger
	MergeTrigger(current, desired *kEventing.Trigger) *kEventing.Trigger

	GetBrokerBindings(
		iface *dtdv0.TwinInterface,
		broker rabbitmqv1beta1.Exchange,
		queue rabbitmqv1beta1.Queue,
	) []rabbitmqv1beta1.Binding
}

type historicalDataStore struct{}

func NewHistoricalDataStore() HistoricalDataStore {
	return &historicalDataStore{}
}

func (h *historicalDataStore) GetService(hds *corev0.HistoricalDataStore) *kserving.Service {
	name := hds.Name
	timeout := strconv.Itoa(*hds.Spec.Timeout)

	ann := map[string]string{}
	if !reflect.DeepEqual(hds.Spec.AutoScaling, corev0.HistoricalDataStoreAutoScaling{}) {
		as := hds.Spec.AutoScaling
		if as.MaxScale != nil {
			ann["autoscaling.knative.dev/maxScale"] = strconv.Itoa(*as.MaxScale)
		}
		if as.MinScale != nil {
			ann["autoscaling.knative.dev/minScale"] = strconv.Itoa(*as.MinScale)
		}
		if as.Target != nil {
			ann["autoscaling.knative.dev/target"] = strconv.Itoa(*as.Target)
		}
		if as.TargetUtilizationPercentage != nil {
			ann["autoscaling.knative.dev/target-utilization-percentage"] = strconv.Itoa(*as.TargetUtilizationPercentage)
		}
		if as.Metric != "" {
			ann["autoscaling.knative.dev/metric"] = string(as.Metric)
		}
	}

	c := corev1.Container{
		Name:            HISTORICAL_DATA_STORE_SERVICE + "-v1",
		Image:           "gcr.io/google-samples/hello-app:1.0",
		ImagePullPolicy: corev1.PullIfNotPresent,
		Resources:       hds.Spec.Resources,
		Env: []corev1.EnvVar{
			{
				Name: "DB_HOST",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: hds.Spec.Postgres.SecretName,
						},
						Key: "host",
					},
				},
			},
			{
				Name:  "DB_PORT",
				Value: strconv.Itoa(int(hds.Spec.Postgres.Port)),
			},
			{
				Name: "DB_NAME",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: hds.Spec.Postgres.SecretName,
						},
						Key: "database",
					},
				},
			},
			{
				Name: "DB_USER",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: hds.Spec.Postgres.SecretName,
						},
						Key: "username",
					},
				},
			},
			{
				Name: "DB_PASSWORD",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: hds.Spec.Postgres.SecretName,
						},
						Key: "password",
					},
				},
			},
			{
				Name:  "TIMEOUT",
				Value: timeout,
			},
		},
	}

	return &kserving.Service{
		TypeMeta: v1.TypeMeta{
			Kind:       "Service",
			APIVersion: "serving.knative.dev/v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: hds.Namespace,
			Labels:    map[string]string{"twinscale/historical-data-store": name},
			OwnerReferences: []v1.OwnerReference{{
				APIVersion: hds.APIVersion,
				Kind:       hds.Kind,
				Name:       name,
				UID:        hds.UID,
			}},
		},
		Spec: kserving.ServiceSpec{
			ConfigurationSpec: kserving.ConfigurationSpec{
				Template: kserving.RevisionTemplateSpec{
					ObjectMeta: v1.ObjectMeta{Annotations: ann},
					Spec: kserving.RevisionSpec{
						PodSpec: corev1.PodSpec{
							NodeSelector: map[string]string{
								"kubernetes.io/arch": "amd64",
								"twinscale-node":     "core",
							},
							Containers: []corev1.Container{c},
						},
					},
				},
			},
		},
	}
}

func (h *historicalDataStore) MergeService(current, desired *kserving.Service) *kserving.Service {
	current.Spec.ConfigurationSpec = desired.Spec.ConfigurationSpec
	return current
}

func (h *historicalDataStore) GetTrigger(hds *corev0.HistoricalDataStore) *kEventing.Trigger {
	name := hds.Name
	namespace := hds.Namespace

	triggerName := fmt.Sprintf("%s-trigger", name)

	return knative.NewTrigger(knative.TriggerParameters{
		TriggerName:    triggerName,
		Namespace:      namespace,
		BrokerName:     BROKER_NAME,
		SubscriberName: name,
		OwnerReferences: []v1.OwnerReference{{
			APIVersion: hds.APIVersion,
			Kind:       hds.Kind,
			Name:       name,
			UID:        hds.UID,
		}},
		Attributes: map[string]string{
			"type": "twinscale.historicalstore",
		},
		Labels: map[string]string{
			"twinscale/historical-data-store": name,
		},
		URL: knative.TriggerURLParameters{
			Path: "/api/v1/historical-events",
		},
		Parallelism:   hds.Spec.AutoScaling.Parallelism,
		CPURequest:    hds.Spec.DispatcherResources.Requests.Cpu().String(),
		MemoryRequest: hds.Spec.DispatcherResources.Requests.Memory().String(),
		CPULimit:      hds.Spec.DispatcherResources.Limits.Cpu().String(),
		MemoryLimit:   hds.Spec.DispatcherResources.Limits.Memory().String(),
	})
}

func (h *historicalDataStore) MergeTrigger(current, desired *kEventing.Trigger) *kEventing.Trigger {
	current.ObjectMeta.Labels = desired.ObjectMeta.Labels
	current.ObjectMeta.Annotations = desired.ObjectMeta.Annotations
	current.Spec = desired.Spec
	return current
}

func (h *historicalDataStore) GetBrokerBindings(
	iface *dtdv0.TwinInterface,
	broker rabbitmqv1beta1.Exchange,
	queue rabbitmqv1beta1.Queue,
) []rabbitmqv1beta1.Binding {
	triggerName := fmt.Sprintf("%s-trigger", iface.Name)

	b, _ := rabbitmq.NewBinding(rabbitmq.BindingArgs{
		Name:      strings.ToLower(iface.Name) + "-historical-store",
		Namespace: iface.Namespace,
		Owner: []v1.OwnerReference{{
			APIVersion: iface.APIVersion,
			Kind:       iface.Kind,
			Name:       iface.Name,
			UID:        iface.UID,
		}},
		RabbitmqClusterReference: &rabbitmqv1beta1.RabbitmqClusterReference{
			Name:      "rabbitmq",
			Namespace: "twinscale",
		},
		RabbitMQVhost: "/",
		Source:        broker.Spec.Name,
		Destination:   queue.Spec.Name,
		Filters: map[string]string{
			"type":              fmt.Sprintf("twinscale.historicalstore.%s", iface.Name),
			"x-knative-trigger": triggerName,
			"x-match":           "all",
		},
		Labels: map[string]string{},
	})

	return []rabbitmqv1beta1.Binding{b}
}
