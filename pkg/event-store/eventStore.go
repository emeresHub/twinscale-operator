package eventStore

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev0 "github.com/emereshub/twinscale-operator/api/core/v0"
	dtdv0 "github.com/emereshub/twinscale-operator/api/dtd/v0"
	"github.com/emereshub/twinscale-operator/pkg/naming"
	knative "github.com/emereshub/twinscale-operator/pkg/third-party/knative"
	"github.com/emereshub/twinscale-operator/pkg/third-party/rabbitmq"

	rabbitmqv1beta1 "github.com/rabbitmq/messaging-topology-operator/api/v1beta1"
	kEventing "knative.dev/eventing/pkg/apis/eventing/v1"
	kserving "knative.dev/serving/pkg/apis/serving/v1"
)

const (
	EVENT_STORE_SERVICE = "twinscale-event-store"
)

func NewEventStore() EventStore {
	return &eventStore{}
}

type EventStore interface {
	GetEventStoreService(eventStore *corev0.EventStore) *kserving.Service
	MergeEventStoreService(currentService *kserving.Service, newService *kserving.Service) *kserving.Service

	GetEventStoreTrigger(eventStore *corev0.EventStore) *kEventing.Trigger
	MergeEventStoreTrigger(currentTrigger *kEventing.Trigger, newTrigger *kEventing.Trigger) *kEventing.Trigger

	GetEventStoreBrokerBindings(twinInterface *dtdv0.TwinInterface, brokerExchange rabbitmqv1beta1.Exchange, eventStoreQueue rabbitmqv1beta1.Queue) []rabbitmqv1beta1.Binding
}

type eventStore struct{}

func (t *eventStore) GetEventStoreService(eventStore *corev0.EventStore) *kserving.Service {
	eventStoreName := eventStore.ObjectMeta.Name
	timeoutValue := fmt.Sprintf("%d", *eventStore.Spec.Timeout)

	var autoScalingAnnotations map[string]string
	if !reflect.DeepEqual(eventStore.Spec.AutoScaling, corev0.EventStoreAutoScaling{}) {
		autoScaling := eventStore.Spec.AutoScaling
		autoScalingAnnotations = make(map[string]string)

		if autoScaling.MaxScale != nil {
			autoScalingAnnotations["autoscaling.knative.dev/maxScale"] = strconv.Itoa(*autoScaling.MaxScale)
		}
		if autoScaling.MinScale != nil {
			autoScalingAnnotations["autoscaling.knative.dev/minScale"] = strconv.Itoa(*autoScaling.MinScale)
		}
		if autoScaling.Target != nil {
			autoScalingAnnotations["autoscaling.knative.dev/target"] = strconv.Itoa(*autoScaling.Target)
		}
		if autoScaling.TargetUtilizationPercentage != nil {
			autoScalingAnnotations["autoscaling.knative.dev/target-utilization-percentage"] = strconv.Itoa(*autoScaling.TargetUtilizationPercentage)
		}
		if autoScaling.Metric != "" {
			autoScalingAnnotations["autoscaling.knative.dev/metric"] = string(autoScaling.Metric)
		}
	}

	return &kserving.Service{
		TypeMeta: v1.TypeMeta{
			Kind:       "Service",
			APIVersion: "serving.knative.dev/v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      eventStoreName,
			Namespace: eventStore.ObjectMeta.Namespace,
			Labels: map[string]string{
				"twinscale/event-store": eventStoreName,
			},
			OwnerReferences: []v1.OwnerReference{
				{
					APIVersion: eventStore.APIVersion,
					Kind:       eventStore.Kind,
					Name:       eventStoreName,
					UID:        eventStore.UID,
				},
			},
		},
		Spec: kserving.ServiceSpec{
			ConfigurationSpec: kserving.ConfigurationSpec{
				Template: kserving.RevisionTemplateSpec{
					ObjectMeta: v1.ObjectMeta{
						Annotations: autoScalingAnnotations,
					},
					Spec: kserving.RevisionSpec{
						PodSpec: corev1.PodSpec{
							NodeSelector: map[string]string{
								"kubernetes.io/arch": "amd64",
								"twinscale-node":     "core",
							},
							Containers: []corev1.Container{
								{
									Name:            EVENT_STORE_SERVICE + "-v1",
									Image:           naming.GetContainerRegistry(EVENT_STORE_SERVICE + ":0.1"),
									ImagePullPolicy: corev1.PullIfNotPresent,
									Resources:       eventStore.Spec.Resources,
									Env: []corev1.EnvVar{
										{
											Name: "DB_HOST",
											ValueFrom: &corev1.EnvVarSource{
												SecretKeyRef: &corev1.SecretKeySelector{
													LocalObjectReference: corev1.LocalObjectReference{
														Name: eventStore.Spec.Postgres.SecretName,
													},
													Key: "host",
												},
											},
										},
										{
											Name:  "DB_PORT",
											Value: strconv.Itoa(int(eventStore.Spec.Postgres.Port)),
										},
										{
											Name: "DB_NAME",
											ValueFrom: &corev1.EnvVarSource{
												SecretKeyRef: &corev1.SecretKeySelector{
													LocalObjectReference: corev1.LocalObjectReference{
														Name: eventStore.Spec.Postgres.SecretName,
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
														Name: eventStore.Spec.Postgres.SecretName,
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
														Name: eventStore.Spec.Postgres.SecretName,
													},
													Key: "password",
												},
											},
										},
										{
											Name:  "TIMEOUT",
											Value: timeoutValue,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (t *eventStore) MergeEventStoreService(currentService, newService *kserving.Service) *kserving.Service {
	currentService.Spec.ConfigurationSpec = newService.Spec.ConfigurationSpec
	return currentService
}

func (t *eventStore) GetEventStoreTrigger(eventStore *corev0.EventStore) *kEventing.Trigger {
	name := eventStore.Name
	namespace := eventStore.Namespace

	var cpuRequest, memoryRequest, cpuLimit, memoryLimit string
	if eventStore.Spec.DispatcherResources.Requests.Cpu() != nil {
		cpuRequest = eventStore.Spec.DispatcherResources.Requests.Cpu().String()
	}
	if eventStore.Spec.DispatcherResources.Requests.Memory() != nil {
		memoryRequest = eventStore.Spec.DispatcherResources.Requests.Memory().String()
	}
	if eventStore.Spec.DispatcherResources.Limits.Cpu() != nil {
		cpuLimit = eventStore.Spec.DispatcherResources.Limits.Cpu().String()
	}
	if eventStore.Spec.DispatcherResources.Limits.Memory() != nil {
		memoryLimit = eventStore.Spec.DispatcherResources.Limits.Memory().String()
	}

	triggerName := fmt.Sprintf("%s-trigger", name)

	return knative.NewTrigger(knative.TriggerParameters{
		TriggerName:    triggerName,
		Namespace:      namespace,
		BrokerName:     "twinscale",
		SubscriberName: name,
		OwnerReferences: []v1.OwnerReference{
			{
				APIVersion: eventStore.APIVersion,
				Kind:       eventStore.Kind,
				Name:       name,
				UID:        eventStore.UID,
			},
		},
		Attributes: map[string]string{
			"type": "twinscale.event-store",
		},
		Labels: map[string]string{
			"twinscale/event-store": name,
		},
		URL: knative.TriggerURLParameters{
			Path: "/api/v1/twin-events",
		},
		Parallelism:   eventStore.Spec.AutoScaling.Parallelism,
		CPURequest:    cpuRequest,
		MemoryRequest: memoryRequest,
		CPULimit:      cpuLimit,
		MemoryLimit:   memoryLimit,
	})
}

func (t *eventStore) MergeEventStoreTrigger(currentTrigger, newTrigger *kEventing.Trigger) *kEventing.Trigger {
	currentTrigger.ObjectMeta.Labels = newTrigger.ObjectMeta.Labels
	currentTrigger.ObjectMeta.Annotations = newTrigger.ObjectMeta.Annotations
	currentTrigger.Spec = newTrigger.Spec
	return currentTrigger
}

func (t *eventStore) GetEventStoreBrokerBindings(twinInterface *dtdv0.TwinInterface, brokerExchange rabbitmqv1beta1.Exchange, eventStoreQueue rabbitmqv1beta1.Queue) []rabbitmqv1beta1.Binding {
	var eventStoreBindings []rabbitmqv1beta1.Binding
	triggerName := fmt.Sprintf("%s-trigger", twinInterface.Name)

	eventStoreEventingBinding, _ := rabbitmq.NewBinding(rabbitmq.BindingArgs{
		Name:      strings.ToLower(twinInterface.Name) + "-event-store",
		Namespace: twinInterface.Namespace,
		Owner: []v1.OwnerReference{
			{
				APIVersion: twinInterface.APIVersion,
				Kind:       twinInterface.Kind,
				Name:       twinInterface.Name,
				UID:        twinInterface.UID,
			},
		},
		RabbitmqClusterReference: &rabbitmqv1beta1.RabbitmqClusterReference{
			Name:      "rabbitmq",
			Namespace: "twinscale",
		},
		RabbitMQVhost: "/",
		Source:        brokerExchange.Spec.Name,
		Destination:   eventStoreQueue.Spec.Name,
		Filters: map[string]string{
			"type":              naming.GetEventTypeEventStoreGenerated(twinInterface.Name),
			"x-knative-trigger": triggerName,
			"x-match":           "all",
		},
		Labels: map[string]string{},
	})
	eventStoreBindings = append(eventStoreBindings, eventStoreEventingBinding)

	return eventStoreBindings
}
