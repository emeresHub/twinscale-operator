package rabbitmq

import (
	"encoding/json"
	"fmt"

	rabbitmqv1beta1 "github.com/rabbitmq/messaging-topology-operator/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	BindingKey    = "x-knative-trigger"
	DLQBindingKey = "x-knative-dlq"
)

type BindingArgs struct {
	Name                     string
	Namespace                string
	Owner                    []metav1.OwnerReference
	RabbitmqClusterReference *rabbitmqv1beta1.RabbitmqClusterReference
	RabbitMQVhost            string
	Source                   string
	Destination              string
	RoutingKey               string
	Filters                  map[string]string
	Labels                   map[string]string
	ClusterName              string
}

func NewBinding(args BindingArgs) (rabbitmqv1beta1.Binding, error) {
	if args.Filters == nil {
		args.Filters = map[string]string{}
	}

	argumentsJson, err := json.Marshal(args.Filters)
	if err != nil {
		return rabbitmqv1beta1.Binding{}, fmt.Errorf("failed to encode binding arguments %+v : %s", argumentsJson, err)
	}

	binding := rabbitmqv1beta1.Binding{
		ObjectMeta: v1.ObjectMeta{
			Name:            args.Name,
			Namespace:       args.Namespace,
			OwnerReferences: args.Owner,
			Labels:          args.Labels,
		},
		Spec: rabbitmqv1beta1.BindingSpec{
			Vhost:           args.RabbitMQVhost,
			Source:          args.Source,
			Destination:     args.Destination,
			DestinationType: "queue",
			RoutingKey:      args.RoutingKey,
			Arguments: &runtime.RawExtension{
				Raw: argumentsJson,
			},
			RabbitmqClusterReference: *args.RabbitmqClusterReference,
		},
	}

	return binding, nil
}
