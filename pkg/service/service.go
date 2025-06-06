package service

import (
	"reflect"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	keventing "knative.dev/eventing/pkg/apis/eventing/v1"
	kserving "knative.dev/serving/pkg/apis/serving/v1"

	dtdv0 "github.com/emereshub/twinscale-operator/api/dtd/v0"
)

// Used to inject settings as environment variables
type twinscaleEnvironmentSettings struct {
	Relationships []twinscaleRelationshipSettings `json:"relationships"`
	Parent        twinscaleRelationshipSettings   `json:"parent"`
}

type twinscaleRelationshipSettings struct {
	Name      string `json:"name"`
	Interface string `json:"interface"`
	Instance  string `json:"instance"`
}

type TwinServiceParameters struct {
	TwinInterface     *dtdv0.TwinInterface
	Broker            keventing.Broker
	EventStoreService kserving.Service
}

func NewTwinService() TwinService {
	return &twinService{}
}

type TwinService interface {
	GetService(twinServiceParameters TwinServiceParameters) *kserving.Service
	MergeTwinService(currentService *kserving.Service, newService *kserving.Service) *kserving.Service
	CompareTwinService(currentService *kserving.Service, newService *kserving.Service) bool
	GetServiceDeletionCriteria(namespacedName types.NamespacedName) map[string]string
}

type twinService struct{}

func (e *twinService) getServiceLabels(twinInterfaceName string) map[string]string {
	return map[string]string{
		"twinscale/twininterface": twinInterfaceName,
	}
}

func (e *twinService) GetServiceDeletionCriteria(namespacedName types.NamespacedName) map[string]string {
	return e.getServiceLabels(namespacedName.Name)
}

func (e *twinService) getTwinInterfaceContainers(twinServiceParameters TwinServiceParameters) []corev1.Container {
	var containers []corev1.Container

	brokerUrl := twinServiceParameters.Broker.Status.Address.URL.URL()
	eventStoreUrl := twinServiceParameters.EventStoreService.Status.URL.URL()

	environmentVariables := []corev1.EnvVar{
		{
			Name:  "TWINSCALE_BROKER",
			Value: brokerUrl.String(),
		},
		{
			Name:  "TWINSCALE_EVENT_STORE",
			Value: eventStoreUrl.String(),
		},
		{
			Name:  "TWINSCALE_GRAPH_URL",
			Value: "http://twinscale-graph-store.twinscale.svc.cluster.local/api/v1/twin-graph",
		},
	}

	for _, container := range twinServiceParameters.TwinInterface.Spec.Service.Template.Spec.Containers {
		container.Env = append(container.Env, environmentVariables...)
		containers = append(containers, container)
	}

	return containers
}

func (t *twinService) GetService(twinServiceParameters TwinServiceParameters) *kserving.Service {
	twinInterface := twinServiceParameters.TwinInterface
	twinInterfaceName := twinInterface.ObjectMeta.Name
	containers := t.getTwinInterfaceContainers(twinServiceParameters)
	var autoScalingAnnotations map[string]string = make(map[string]string)

	if !reflect.DeepEqual(twinInterface.Spec.Service.AutoScaling, dtdv0.TwinInterfaceAutoScaling{}) {
		autoScaling := twinInterface.Spec.Service.AutoScaling
		autoScalingAnnotations = make(map[string]string)
		if autoScaling.MaxScale != nil {
			autoScalingAnnotations["autoscaling.knative.dev/maxScale"] = strconv.Itoa(*autoScaling.MaxScale)
		} else {
			autoScalingAnnotations["autoscaling.knative.dev/maxScale"] = strconv.Itoa(1)
		}

		if autoScaling.MinScale != nil {
			autoScalingAnnotations["autoscaling.knative.dev/minScale"] = strconv.Itoa(*autoScaling.MinScale)
		} else {
			autoScalingAnnotations["autoscaling.knative.dev/minScale"] = strconv.Itoa(0)
		}

		if autoScaling.Target != nil {
			autoScalingAnnotations["autoscaling.knative.dev/target"] = strconv.Itoa(*autoScaling.Target)
		}

		if autoScaling.TargetUtilizationPercentage != nil {
			autoScalingAnnotations["autoscaling.knative.dev/target-utilization-percentage"] = strconv.Itoa(*autoScaling.TargetUtilizationPercentage)
		}

		if autoScaling.Metric != "" {
			autoScalingAnnotations["autoscaling.knative.dev/metric"] = string(*&autoScaling.Metric)
		}
	}

	service := &kserving.Service{
		TypeMeta: v1.TypeMeta{
			Kind:       "Service",
			APIVersion: "serving.knative.dev/v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      twinInterface.ObjectMeta.Name,
			Namespace: twinInterface.ObjectMeta.Namespace,
			Labels:    t.getServiceLabels(twinInterfaceName),
			OwnerReferences: []v1.OwnerReference{
				{
					APIVersion: twinInterface.APIVersion,
					Kind:       twinInterface.Kind,
					Name:       twinInterface.Name,
					UID:        twinInterface.UID,
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
								"twinscale-node":     "service",
							},
							Containers: containers,
						},
					},
				},
			},
		},
	}
	return service
}

func (t *twinService) MergeTwinService(currentService *kserving.Service, newService *kserving.Service) *kserving.Service {
	currentService.Spec.ConfigurationSpec = newService.Spec.ConfigurationSpec
	return currentService
}

// Compare Twin Services.
// If no changes between current and new, return true.
// If some change was identified between current and new, return false.
func (t *twinService) CompareTwinService(currentService *kserving.Service, newService *kserving.Service) bool {
	newAnnotations := newService.Spec.ConfigurationSpec.Template.ObjectMeta.Annotations
	currentAnnotations := currentService.Spec.ConfigurationSpec.Template.ObjectMeta.Annotations

	if !reflect.DeepEqual(currentAnnotations, newAnnotations) {
		return false
	}

	newContainers := newService.Spec.ConfigurationSpec.Template.Spec.Containers
	currentContainers := currentService.Spec.ConfigurationSpec.Template.Spec.Containers

	if len(newContainers) != len(currentContainers) {
		return false
	}

	for i, newContainer := range newContainers {
		currentContainer := currentContainers[i]

		if newContainer.Image != currentContainer.Image {
			return false
		}

		if !reflect.DeepEqual(newContainer.Resources, currentContainer.Resources) {
			return false
		}
	}

	return true
}
