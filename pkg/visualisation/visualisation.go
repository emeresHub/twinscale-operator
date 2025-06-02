// pkg/visualisation/visualisation.go

package visualisation

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
	VISUALISATION_SERVICE = "twinscale-visualisation"
	BROKER_NAME           = "twinscale"
)

// Visualisation defines the interface for building Knative Service, Trigger, and RabbitMQ bindings
type Visualisation interface {
	GetService(vis *corev0.Visualisation) *kserving.Service
	MergeService(current, desired *kserving.Service) *kserving.Service

	GetTrigger(vis *corev0.Visualisation) *kEventing.Trigger
	MergeTrigger(current, desired *kEventing.Trigger) *kEventing.Trigger

	// Now accepts both the Visualisation and the TwinInterface so we can use vis.Name:
	GetBrokerBindings(
		vis *corev0.Visualisation,
		iface *dtdv0.TwinInterface,
		broker rabbitmqv1beta1.Exchange,
		queue rabbitmqv1beta1.Queue,
	) []rabbitmqv1beta1.Binding
}

type visualisation struct{}

func NewVisualisation() Visualisation {
	return &visualisation{}
}

func (v *visualisation) GetService(vis *corev0.Visualisation) *kserving.Service {
	name := vis.Name
	timeout := "0"
	if vis.Spec.Timeout != nil {
		timeout = strconv.Itoa(*vis.Spec.Timeout)
	}

	// Build autoscaling annotations if provided
	ann := map[string]string{}
	if !reflect.DeepEqual(vis.Spec.AutoScaling, corev0.VisualisationAutoScaling{}) {
		as := vis.Spec.AutoScaling
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
		// Parallelism (RabbitMQ-specific) if provided:
		if as.Parallelism != nil {
			ann["rabbitmq.eventing.knative.dev/parallelism"] = strconv.Itoa(*as.Parallelism)
		}
	}

	// Container spec
	c := corev1.Container{
		Name:            VISUALISATION_SERVICE + "-v1",
		Image:           naming.GetContainerRegistry(VISUALISATION_SERVICE + ":0.1"),
		ImagePullPolicy: corev1.PullIfNotPresent,
		Resources:       vis.Spec.Resources,
		Env: []corev1.EnvVar{{
			Name:  "TIMEOUT",
			Value: timeout,
		}},
	}

	return &kserving.Service{
		TypeMeta: v1.TypeMeta{
			Kind:       "Service",
			APIVersion: "serving.knative.dev/v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: vis.Namespace,
			Labels:    map[string]string{"twinscale/visualisation": name},
			OwnerReferences: []v1.OwnerReference{{
				APIVersion: vis.APIVersion,
				Kind:       vis.Kind,
				Name:       name,
				UID:        vis.UID,
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
								"twinscale-node":     "visualisation",
							},
							Containers: []corev1.Container{c},
						},
					},
				},
			},
		},
	}
}

func (v *visualisation) MergeService(current, desired *kserving.Service) *kserving.Service {
	current.Spec.ConfigurationSpec = desired.Spec.ConfigurationSpec
	return current
}

func (v *visualisation) GetTrigger(vis *corev0.Visualisation) *kEventing.Trigger {
	triggerName := fmt.Sprintf("%s-trigger", vis.Name)

	return knative.NewTrigger(knative.TriggerParameters{
		TriggerName:    triggerName,
		Namespace:      vis.Namespace,
		BrokerName:     BROKER_NAME,
		SubscriberName: VISUALISATION_SERVICE,
		OwnerReferences: []v1.OwnerReference{{
			APIVersion: vis.APIVersion,
			Kind:       vis.Kind,
			Name:       vis.Name,
			UID:        vis.UID,
		}},
		Attributes: map[string]string{
			"type": "twinscale.visualisation." + vis.Name,
		},
		Labels: map[string]string{
			"twinscale/visualisation": VISUALISATION_SERVICE,
		},
		URL: knative.TriggerURLParameters{
			Path: "/api/v1/visualisation-events",
		},
		Parallelism:   vis.Spec.AutoScaling.Parallelism,
		CPURequest:    vis.Spec.DispatcherResources.Requests.Cpu().String(),
		MemoryRequest: vis.Spec.DispatcherResources.Requests.Memory().String(),
		CPULimit:      vis.Spec.DispatcherResources.Limits.Cpu().String(),
		MemoryLimit:   vis.Spec.DispatcherResources.Limits.Memory().String(),
	})
}

func (v *visualisation) MergeTrigger(current, desired *kEventing.Trigger) *kEventing.Trigger {
	current.ObjectMeta.Annotations = desired.ObjectMeta.Annotations
	return current
}

// Now we pass in *corev0.Visualisation so we can use vis.Name:
func (v *visualisation) GetBrokerBindings(
	vis *corev0.Visualisation,
	iface *dtdv0.TwinInterface,
	broker rabbitmqv1beta1.Exchange,
	queue rabbitmqv1beta1.Queue,
) []rabbitmqv1beta1.Binding {
	// Use vis.Name when constructing x-knative-trigger:
	triggerName := fmt.Sprintf("%s-trigger", vis.Name)

	bindingName := strings.ToLower(iface.Name) + "-visualisation"
	b, _ := rabbitmq.NewBinding(rabbitmq.BindingArgs{
		Name:      bindingName,
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
			"type":              fmt.Sprintf("twinscale.visualisation.%s", iface.Name),
			"x-knative-trigger": triggerName,
			"x-match":           "all",
		},
		Labels: map[string]string{},
	})

	return []rabbitmqv1beta1.Binding{b}
}
