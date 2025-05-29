package knative

import (
	"reflect"
	"strconv"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kEventing "knative.dev/eventing/pkg/apis/eventing/v1"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

// https://github.com/knative-extensions/eventing-rabbitmq/blob/main/pkg/utils/resource_requirements.go#L30C1-L33C72
const (
	ParallelismAnnotation   = "rabbitmq.eventing.knative.dev/parallelism"
	CPURequestAnnotation    = "rabbitmq.eventing.knative.dev/cpu-request"
	CPULimitAnnotation      = "rabbitmq.eventing.knative.dev/cpu-limit"
	MemoryRequestAnnotation = "rabbitmq.eventing.knative.dev/memory-request"
	MemoryLimitAnnotation   = "rabbitmq.eventing.knative.dev/memory-limit"
)

type TriggerParameters struct {
	TriggerName     string
	Namespace       string
	BrokerName      string
	SubscriberName  string
	OwnerReferences []v1.OwnerReference
	Attributes      map[string]string
	Labels          map[string]string
	URL             TriggerURLParameters
	Parallelism     *int
	CPURequest      string
	CPULimit        string
	MemoryRequest   string
	MemoryLimit     string
}

type TriggerURLParameters struct {
	Path string
}

func NewTrigger(triggerParameters TriggerParameters) *kEventing.Trigger {
	var triggerAnnotations = make(map[string]string)

	if triggerParameters.Parallelism != nil {
		triggerAnnotations[ParallelismAnnotation] = strconv.Itoa(*triggerParameters.Parallelism)
	}

	if triggerParameters.MemoryLimit != "" {
		triggerAnnotations[MemoryLimitAnnotation] = triggerParameters.MemoryLimit
	}

	if triggerParameters.MemoryRequest != "" {
		triggerAnnotations[MemoryRequestAnnotation] = triggerParameters.MemoryRequest
	}

	if triggerParameters.CPULimit != "" {
		triggerAnnotations[CPULimitAnnotation] = triggerParameters.CPULimit
	}

	if triggerParameters.CPURequest != "" {
		triggerAnnotations[CPURequestAnnotation] = triggerParameters.CPURequest
	}

	var urlParameters *apis.URL
	if !reflect.DeepEqual(triggerParameters.URL, TriggerURLParameters{}) {
		urlParameters = &apis.URL{
			Path: triggerParameters.URL.Path,
		}
	}

	return &kEventing.Trigger{
		TypeMeta: v1.TypeMeta{
			Kind:       "Trigger",
			APIVersion: "eventing.knative.dev/v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:            triggerParameters.TriggerName,
			Namespace:       triggerParameters.Namespace,
			Labels:          triggerParameters.Labels,
			OwnerReferences: triggerParameters.OwnerReferences,
			Annotations:     triggerAnnotations,
		},
		Spec: kEventing.TriggerSpec{
			Broker: triggerParameters.BrokerName,
			Filter: &kEventing.TriggerFilter{
				Attributes: triggerParameters.Attributes,
			},
			Subscriber: duckv1.Destination{
				Ref: &duckv1.KReference{
					Kind:       "Service",
					APIVersion: "serving.knative.dev/v1",
					Name:       triggerParameters.SubscriberName,
				},
				URI: urlParameters,
			},
		},
	}
}
