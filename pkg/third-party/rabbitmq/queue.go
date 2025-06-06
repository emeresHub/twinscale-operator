package rabbitmq

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	rabbitmqv1beta1 "github.com/rabbitmq/messaging-topology-operator/api/v1beta1"
)

type QueueArgs struct {
	Name                     string
	Namespace                string
	RabbitMQVhost            string
	QueueName                string
	RabbitmqClusterReference *rabbitmqv1beta1.RabbitmqClusterReference
	Owner                    metav1.OwnerReference
	Labels                   map[string]string
	DLXName                  *string
	BrokerUID                string
}

func NewQueue(args *QueueArgs) *rabbitmqv1beta1.Queue {
	queueName := args.Name
	if args.QueueName != "" {
		queueName = args.QueueName
	}

	return &rabbitmqv1beta1.Queue{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       args.Namespace,
			Name:            args.Name,
			OwnerReferences: []metav1.OwnerReference{args.Owner},
			Labels:          args.Labels,
		},
		Spec: rabbitmqv1beta1.QueueSpec{
			Name:                     queueName,
			Vhost:                    args.RabbitMQVhost,
			Durable:                  true,
			AutoDelete:               false,
			RabbitmqClusterReference: *args.RabbitmqClusterReference,
			Type:                     "quorum",
		},
	}
}
