/*
Copyright 2025 emereshub.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package core

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	corev1 "k8s.io/api/core/v1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	corev0 "github.com/emereshub/twinscale-operator/api/core/v0"
	"github.com/emereshub/twinscale-operator/pkg/event"
	"github.com/emereshub/twinscale-operator/pkg/naming"
	"github.com/emereshub/twinscale-operator/pkg/third-party/rabbitmq"

	rabbitmqv1beta1 "github.com/rabbitmq/messaging-topology-operator/api/v1beta1"
)

// MQTTTriggerReconciler reconciles a MQTTTrigger object
type MQTTTriggerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=core.core.twinscale.io,resources=mqtttriggers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.core.twinscale.io,resources=mqtttriggers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.core.twinscale.io,resources=mqtttriggers/finalizers,verbs=update

func (r *MQTTTriggerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the MQTTTrigger resource
	var mqttTrigger corev0.MQTTTrigger
	if err := r.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: req.Name}, &mqttTrigger); err != nil {
		if errors.IsNotFound(err) {
			// CR deleted – nothing to do
			return ctrl.Result{}, nil
		}
		logger.Error(err, "unable to fetch MQTTTrigger", "name", req.Name)
		return ctrl.Result{}, err
	}

	// Create or update all underlying resources
	if _, err := r.createOrUpdateMQTTTrigger(ctx, &mqttTrigger); err != nil {
		// update status on failure
		mqttTrigger.Status.Phase = "Error"
		_ = r.Status().Update(ctx, &mqttTrigger)
		return ctrl.Result{}, err
	}

	// mark ready
	mqttTrigger.Status.Phase = "Ready"
	if err := r.Status().Update(ctx, &mqttTrigger); err != nil {
		logger.Error(err, "unable to update MQTTTrigger status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *MQTTTriggerReconciler) createOrUpdateMQTTTrigger(ctx context.Context, mqttTrigger *corev0.MQTTTrigger) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// 1) Fetch RabbitMQ default-user secret
	var rabbitMQSecret corev1.Secret
	if err := r.Get(ctx, types.NamespacedName{Name: "rabbitmq-default-user", Namespace: "twinscale"}, &rabbitMQSecret); err != nil {
		logger.Error(err, "getting rabbitmq-default-user secret")
		return ctrl.Result{}, err
	}

	// 2) List broker exchanges
	var exchangeList rabbitmqv1beta1.ExchangeList
	if err := r.List(ctx, &exchangeList,
		client.InNamespace("twinscale"),
		client.MatchingLabels(map[string]string{"eventing.knative.dev/broker": "twinscale"}),
	); err != nil {
		logger.Error(err, "listing rabbitmq broker exchanges")
		return ctrl.Result{}, err
	}
	if len(exchangeList.Items) == 0 {
		err := fmt.Errorf("no default broker exchange found")
		logger.Error(err, "broker exchange list empty")
		return ctrl.Result{}, err
	}
	defaultExchange := exchangeList.Items[0]

	// 3) Ensure MQTT dispatcher Queue
	desiredQueue := r.getMQTTDispatcherQueue(mqttTrigger)
	if err := r.applyQueue(ctx, desiredQueue); err != nil {
		return ctrl.Result{}, err
	}

	// 4) Ensure MQTT dispatcher Deployment
	desiredMQTTDep := r.getMQTTDispatcherDeployment(mqttTrigger, &rabbitMQSecret, &defaultExchange)
	if err := r.applyDeployment(ctx, desiredMQTTDep); err != nil {
		return ctrl.Result{}, err
	}

	// 5) Ensure MQTT dispatcher Service
	desiredMQTTSvc := r.getMQTTDispatcherService(mqttTrigger)
	if err := r.applyService(ctx, desiredMQTTSvc); err != nil {
		return ctrl.Result{}, err
	}

	// 6) Ensure CloudEvent dispatcher Queue
	desiredCEQueue := r.getCloudEventDispatcherQueue(mqttTrigger)
	if err := r.applyQueue(ctx, desiredCEQueue); err != nil {
		return ctrl.Result{}, err
	}

	// 7) Ensure CloudEvent dispatcher Deployment
	desiredCEDep := r.getCloudEventDispatcherDeployment(mqttTrigger, &rabbitMQSecret)
	if err := r.applyDeployment(ctx, desiredCEDep); err != nil {
		return ctrl.Result{}, err
	}

	// 8) Ensure CloudEvent dispatcher Service
	desiredCESvc := r.getCloudEventDispatcherService(mqttTrigger)
	if err := r.applyService(ctx, desiredCESvc); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// applyQueue creates or updates a rabbitmqv1beta1.Queue
func (r *MQTTTriggerReconciler) applyQueue(ctx context.Context, desired *rabbitmqv1beta1.Queue) error {
	var existing rabbitmqv1beta1.Queue
	err := r.Get(ctx, types.NamespacedName{Namespace: desired.Namespace, Name: desired.Name}, &existing)
	if errors.IsNotFound(err) {
		return r.Create(ctx, desired)
	} else if err != nil {
		return err
	}
	// merge spec
	existing.Spec = desired.Spec
	return r.Update(ctx, &existing)
}

// applyDeployment creates or updates an apps/v1 Deployment
func (r *MQTTTriggerReconciler) applyDeployment(ctx context.Context, desired *appsv1.Deployment) error {
	var existing appsv1.Deployment
	err := r.Get(ctx, types.NamespacedName{Namespace: desired.Namespace, Name: desired.Name}, &existing)
	if errors.IsNotFound(err) {
		return r.Create(ctx, desired)
	} else if err != nil {
		return err
	}
	existing.Spec = desired.Spec
	return r.Update(ctx, &existing)
}

// applyService creates or updates a core/v1 Service
func (r *MQTTTriggerReconciler) applyService(ctx context.Context, desired *corev1.Service) error {
	var existing corev1.Service
	err := r.Get(ctx, types.NamespacedName{Namespace: desired.Namespace, Name: desired.Name}, &existing)
	if errors.IsNotFound(err) {
		return r.Create(ctx, desired)
	} else if err != nil {
		return err
	}
	// Only merge Spec (not cluster-assigned fields)
	existing.Spec = desired.Spec
	return r.Update(ctx, &existing)
}

// getMQTTDispatcherQueue builds the Queue resource
func (r *MQTTTriggerReconciler) getMQTTDispatcherQueue(trigger *corev0.MQTTTrigger) *rabbitmqv1beta1.Queue {
	args := &rabbitmq.QueueArgs{
		Name:      event.MQTT_DISPATCHER_QUEUE,
		Namespace: trigger.Namespace,
		QueueName: event.MQTT_DISPATCHER_QUEUE,
		RabbitMQVhost: "/",
		RabbitmqClusterReference: &rabbitmqv1beta1.RabbitmqClusterReference{
			Name:      "rabbitmq",
			Namespace: trigger.Namespace,
		},
		Owner: metav1.OwnerReference{
			APIVersion: trigger.APIVersion,
			Kind:       trigger.Kind,
			Name:       trigger.Name,
			UID:        trigger.UID,
		},
	}
	return rabbitmq.NewQueue(args)
}

// getMQTTDispatcherDeployment builds the Deployment for MQTT dispatcher
func (r *MQTTTriggerReconciler) getMQTTDispatcherDeployment(trigger *corev0.MQTTTrigger, secret *corev1.Secret, exchange *rabbitmqv1beta1.Exchange) *appsv1.Deployment {
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mqtt-dispatcher",
			Namespace: trigger.Namespace,
			Labels:    map[string]string{"twinscale/trigger": "mqtt-dispatcher"},
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: trigger.APIVersion,
				Kind:       trigger.Kind,
				Name:       trigger.Name,
				UID:        trigger.UID,
			}},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"twinscale/trigger": "mqtt-dispatcher"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"twinscale/trigger": "mqtt-dispatcher"}},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:            "mqtt-dispatcher",
						Image:           naming.GetContainerRegistry("twinscale-mqtt-dispatcher:0.1"),
						ImagePullPolicy: corev1.PullAlways,
						Env: []corev1.EnvVar{
							{Name: "SERVICE_NAME", Value: event.MQTT_DISPATCHER + "-1"},
							{Name: "PROTOCOL", Value: "amqp"},
							{Name: "SERVER_URL", Value: string(secret.Data["host"])},
							{Name: "SERVER_PORT", Value: string(secret.Data["port"])},
							{Name: "USERNAME", Value: string(secret.Data["username"])},
							{Name: "PASSWORD", Value: string(secret.Data["password"])},
							{Name: "DECLARE_QUEUE", Value: "false"},
							{Name: "DECLARE_EXCHANGE", Value: "false"},
							{Name: "PUBLISHER_EXCHANGE", Value: exchange.Name},
							{Name: "SUBSCRIBER_QUEUE", Value: event.MQTT_DISPATCHER_QUEUE},
						},
						Ports: []corev1.ContainerPort{
							{ContainerPort: 5672}, {Name: "http-metrics", ContainerPort: 8080},
						},
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
						},
					}},
				},
			},
		},
	}
	return dep
}

// getMQTTDispatcherService builds the Service for MQTT dispatcher
func (r *MQTTTriggerReconciler) getMQTTDispatcherService(trigger *corev0.MQTTTrigger) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      event.MQTT_DISPATCHER,
			Namespace: trigger.Namespace,
			Labels:    map[string]string{"twinscale/trigger": event.MQTT_DISPATCHER},
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: trigger.APIVersion,
				Kind:       trigger.Kind,
				Name:       trigger.Name,
				UID:        trigger.UID,
			}},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"twinscale/trigger": event.MQTT_DISPATCHER},
			Ports: []corev1.ServicePort{
				{Name: "mqtt-port", Port: 5672, TargetPort: intstr.FromInt(5672), Protocol: corev1.ProtocolTCP},
				{Name: "http-metrics", Port: 8080, TargetPort: intstr.FromInt(8080), Protocol: corev1.ProtocolTCP},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}
}

// getCloudEventDispatcherQueue builds the Queue for CloudEvent dispatcher
func (r *MQTTTriggerReconciler) getCloudEventDispatcherQueue(trigger *corev0.MQTTTrigger) *rabbitmqv1beta1.Queue {
	args := &rabbitmq.QueueArgs{
		Name:      event.CLOUD_EVENT_DISPATCHER_QUEUE,
		Namespace: trigger.Namespace,
		QueueName: event.CLOUD_EVENT_DISPATCHER_QUEUE,
		RabbitMQVhost: "/",
		RabbitmqClusterReference: &rabbitmqv1beta1.RabbitmqClusterReference{
			Name:      "rabbitmq",
			Namespace: trigger.Namespace,
		},
		Owner: metav1.OwnerReference{
			APIVersion: trigger.APIVersion,
			Kind:       trigger.Kind,
			Name:       trigger.Name,
			UID:        trigger.UID,
		},
	}
	return rabbitmq.NewQueue(args)
}

// getCloudEventDispatcherDeployment builds the Deployment for CloudEvent dispatcher
func (r *MQTTTriggerReconciler) getCloudEventDispatcherDeployment(trigger *corev0.MQTTTrigger, secret *corev1.Secret) *appsv1.Deployment {
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      event.CLOUD_EVENT_DISPATCHER,
			Namespace: trigger.Namespace,
			Labels:    map[string]string{"twinscale/trigger": event.CLOUD_EVENT_DISPATCHER},
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: trigger.APIVersion,
				Kind:       trigger.Kind,
				Name:       trigger.Name,
				UID:        trigger.UID,
			}},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"twinscale/trigger": event.CLOUD_EVENT_DISPATCHER}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"twinscale/trigger": event.CLOUD_EVENT_DISPATCHER}},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:            event.CLOUD_EVENT_DISPATCHER,
						Image:           naming.GetContainerRegistry("twinscale-cloud-event-dispatcher:0.1"),
						ImagePullPolicy: corev1.PullAlways,
						Env: []corev1.EnvVar{
							{Name: "SERVICE_NAME", Value: event.CLOUD_EVENT_DISPATCHER + "-1"},
							{Name: "PROTOCOL", Value: "amqp"},
							{Name: "SERVER_URL", Value: string(secret.Data["host"])},
							{Name: "SERVER_PORT", Value: string(secret.Data["port"])},
							{Name: "USERNAME", Value: string(secret.Data["username"])},
							{Name: "PASSWORD", Value: string(secret.Data["password"])},
							{Name: "DECLARE_QUEUE", Value: "false"},
							{Name: "DECLARE_EXCHANGE", Value: "false"},
							{Name: "PUBLISHER_EXCHANGE", Value: event.CLOUD_EVENT_DISPATCHER_EXCHANGE},
							{Name: "SUBSCRIBER_QUEUE", Value: event.CLOUD_EVENT_DISPATCHER_QUEUE},
						},
						Ports: []corev1.ContainerPort{{ContainerPort: 5672}},
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("128Mi"),
							},
						},
					}},
				},
			},
		},
	}
	return dep
}

// getCloudEventDispatcherService builds the Service for CloudEvent dispatcher
func (r *MQTTTriggerReconciler) getCloudEventDispatcherService(trigger *corev0.MQTTTrigger) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      event.CLOUD_EVENT_DISPATCHER,
			Namespace: trigger.Namespace,
			Labels:    map[string]string{"twinscale/trigger": event.CLOUD_EVENT_DISPATCHER},
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: trigger.APIVersion,
				Kind:       trigger.Kind,
				Name:       trigger.Name,
				UID:        trigger.UID,
			}},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"twinscale/trigger": event.CLOUD_EVENT_DISPATCHER},
			Ports:    []corev1.ServicePort{{Port: 5672, TargetPort: intstr.FromInt(5672), Protocol: corev1.ProtocolTCP}},
			Type:     corev1.ServiceTypeClusterIP,
		},
	}
}

func int32Ptr(i int32) *int32 { return &i }

// SetupWithManager sets up the controller with the Manager.
func (r *MQTTTriggerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev0.MQTTTrigger{}).
		Owns(&rabbitmqv1beta1.Queue{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
