package rabbitmq

import (
	"net/url"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/kmeta"

	rabbitmqv1beta1 "github.com/rabbitmq/messaging-topology-operator/api/v1beta1"
	keventing "knative.dev/eventing/pkg/apis/eventing/v1"
)

type ExchangeArgs struct {
	Name                     string
	Namespace                string
	RabbitMQVhost            string
	RabbitmqClusterReference *rabbitmqv1beta1.RabbitmqClusterReference
	Labels                   map[string]string
	RabbitMQURL              *url.URL
	Broker                   *keventing.Broker
	Trigger                  *keventing.Trigger
}

func NewExchange(args *ExchangeArgs) *rabbitmqv1beta1.Exchange {
	// exchange configurations for triggers and broker
	var exchangeName string
	var ownerReference metav1.OwnerReference
	if args.Trigger != nil {
		ownerReference = *kmeta.NewControllerRef(args.Trigger)
		exchangeName = args.Name
	} else if args.Broker != nil {
		ownerReference = *kmeta.NewControllerRef(args.Broker)
		exchangeName = args.Name
	}

	return &rabbitmqv1beta1.Exchange{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       args.Namespace,
			Name:            args.Name,
			OwnerReferences: []metav1.OwnerReference{ownerReference},
			Labels:          args.Labels,
		},
		Spec: rabbitmqv1beta1.ExchangeSpec{
			Name:                     exchangeName,
			Vhost:                    args.RabbitMQVhost,
			Type:                     "headers",
			Durable:                  true,
			AutoDelete:               false,
			RabbitmqClusterReference: *args.RabbitmqClusterReference,
		},
	}
}
