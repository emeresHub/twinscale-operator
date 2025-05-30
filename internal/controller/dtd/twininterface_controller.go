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

package dtd

import (
	"context"
	"fmt"

	rabbitmqv1beta1 "github.com/rabbitmq/messaging-topology-operator/api/v1beta1"
	eventingv1 "knative.dev/eventing/pkg/apis/eventing/v1"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	dtdv0 "github.com/emereshub/twinscale-operator/api/dtd/v0"
	twinevent "github.com/emereshub/twinscale-operator/pkg/event"
	eventStore "github.com/emereshub/twinscale-operator/pkg/event-store"
	twinservice "github.com/emereshub/twinscale-operator/pkg/service"
	kserving "knative.dev/serving/pkg/apis/serving/v1"
)

// TwinInterfaceReconciler reconciles a TwinInterface object
type TwinInterfaceReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	TwinService twinservice.TwinService
	TwinEvent   twinevent.TwinEvent
	EventStore  eventStore.EventStore
}

//+kubebuilder:rbac:groups=dtd.core.twinscale.io,resources=twininterfaces,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dtd.core.twinscale.io,resources=twininterfaces/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dtd.core.twinscale.io,resources=twininterfaces/finalizers,verbs=update

func (r *TwinInterfaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// ensure our helper interfaces are initialized
	if r.TwinService == nil {
		r.TwinService = twinservice.NewTwinService()
	}
	if r.TwinEvent == nil {
		r.TwinEvent = twinevent.NewTwinEvent()
	}
	if r.EventStore == nil {
		r.EventStore = eventStore.NewEventStore()
	}

	twinInterface := &dtdv0.TwinInterface{}
	err := r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, twinInterface)

	// Delete scenario
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.Error(err, fmt.Sprintf("Unexpected error while deleting TwinInstance %s", req.Name))
		return ctrl.Result{}, err
	}

	return r.createUpdateTwinInterface(ctx, req, twinInterface)
}

func (r *TwinInterfaceReconciler) createUpdateTwinInterface(ctx context.Context, req ctrl.Request, twinInterface *dtdv0.TwinInterface) (ctrl.Result, error) {
	twinInterfaceName := twinInterface.ObjectMeta.Name

	var resultErrors []error
	var twinInterfaceTrigger *eventingv1.Trigger
	logger := log.FromContext(ctx)

	// Create Service Instance and Trigger, if pod is specified
	if twinInterface.Spec.Service != nil {
		// Get Broker
		broker := eventingv1.Broker{}
		err := r.Get(ctx, types.NamespacedName{Namespace: "twinscale", Name: "twinscale"}, &broker)
		if err != nil {
			logger.Error(err, "Error while getting Broker")
			resultErrors = append(resultErrors, err)
		}

		// Get Event Store
		eventStoreService := servingv1.Service{}
		err = r.Get(ctx, types.NamespacedName{Namespace: "twinscale", Name: "event-store"}, &eventStoreService)
		if err != nil {
			logger.Error(err, "Error while getting event store")
			resultErrors = append(resultErrors, err)
		}

		newKService := r.TwinService.GetService(twinservice.TwinServiceParameters{
			TwinInterface:     twinInterface,
			Broker:            broker,
			EventStoreService: eventStoreService,
		})

		err = r.Create(ctx, newKService, &client.CreateOptions{})
		if err != nil && !errors.IsAlreadyExists(err) {
			logger.Error(err, fmt.Sprintf("Error while creating Twin Interface Service %s", twinInterfaceName))
			resultErrors = append(resultErrors, err)
		} else if err != nil {
			currentKService := &kserving.Service{}
			err := r.Get(ctx, types.NamespacedName{Namespace: twinInterface.Namespace, Name: twinInterface.Name}, currentKService)
			if err != nil {
				logger.Error(err, fmt.Sprintf("Error while getting current Twin Interface service %s", twinInterface.Name))
				return ctrl.Result{}, err
			}

			twinServiceEqual := r.TwinService.CompareTwinService(currentKService, newKService)
			if !twinServiceEqual {
				currentKService = r.TwinService.MergeTwinService(currentKService, newKService)
				err = r.Update(ctx, currentKService, &client.UpdateOptions{})
				if err != nil {
					logger.Error(err, fmt.Sprintf("Error while updating Twin Interface Service %s", twinInterfaceName))
					resultErrors = append(resultErrors, err)
				}
			} else {
				logger.Info(fmt.Sprintf("No changes to Twin Interface Service: %s", twinInterfaceName))
			}
		}

		// Create Trigger
		twinInterfaceTrigger = r.TwinEvent.GetTwinInterfaceTrigger(twinInterface)
		logger.Info(fmt.Sprintf("Creating Twin Interface Trigger %s", twinInterfaceTrigger.Name))
		err = r.Create(ctx, twinInterfaceTrigger, &client.CreateOptions{})
		if err != nil && !errors.IsAlreadyExists(err) {
			logger.Error(err, fmt.Sprintf("Error while creating Twin Interface Trigger %s", twinInterfaceName))
			resultErrors = append(resultErrors, err)
		}
	}

	// Create MQTT Binding Rules
	bindings := r.TwinEvent.GetMQQTDispatcherBindings(twinInterface)
	for _, binding := range bindings {
		logger.Info(fmt.Sprintf("Creating Twin Interface MQTT Dispatcher Trigger Binding %s", binding.Name))
		err := r.Create(ctx, &binding, &client.CreateOptions{})
		if err != nil && !errors.IsAlreadyExists(err) {
			logger.Error(err, fmt.Sprintf("Error while creating Twin Interface MQTT Dispatcher Trigger Binding %s", binding.Name))
			resultErrors = append(resultErrors, err)
		}
	}

	// Create Relationship RabbitMQ bindings to existing Queue and Eventing
	eventStoreQueue, err := r.getEventStoreQueue(ctx, twinInterface)
	if err != nil {
		logger.Error(err, fmt.Sprintf("No Queue found for event store %s", twinInterfaceName))
		return ctrl.Result{}, err
	}

	brokerExchange, err := r.getBrokerExchange(ctx, req, twinInterface)
	if err != nil {
		logger.Error(err, fmt.Sprintf("No Broker Exchange found for TwinInterface %s", twinInterfaceName))
		resultErrors = append(resultErrors, err)
	} else {

		// EventStore bindings
		bindings := r.EventStore.GetEventStoreBrokerBindings(twinInterface, brokerExchange, eventStoreQueue)
		for _, b3 := range bindings {
			logger.Info(fmt.Sprintf("Creating Twin Command Event Store Binding %s", b3.Name))
			err = r.Create(ctx, &b3, &client.CreateOptions{})
			if err != nil && !errors.IsAlreadyExists(err) {
				logger.Error(err, fmt.Sprintf("Error while creating EventStore binding %s", b3.Name))
				resultErrors = append(resultErrors, err)
			}
		}

		if twinInterfaceTrigger != nil {
			twinInterfaceQueue, err := r.getTwinInterfaceQueue(ctx, req, twinInterface)
			if err != nil {
				if errors.IsNotFound(err) {
					logger.Info(fmt.Sprintf("No Queue found for TwinInterface %s. Requeueing request...", twinInterfaceName))
					return ctrl.Result{Requeue: true}, err
				}
				logger.Error(err, fmt.Sprintf("Error while getting TwinInterface %s Queue", twinInterfaceName))
				resultErrors = append(resultErrors, err)
			} else {
				// Relationship bindings
				bindings := r.TwinEvent.GetRelationshipBrokerBindings(twinInterface, brokerExchange, twinInterfaceQueue)
				for _, b4 := range bindings {
					logger.Info(fmt.Sprintf("Creating Relationship Binding %s", b4.Name))
					err = r.Create(ctx, &b4, &client.CreateOptions{})
					if err != nil && !errors.IsAlreadyExists(err) {
						logger.Error(err, fmt.Sprintf("Error while creating binding %s", b4.Name))
						resultErrors = append(resultErrors, err)
					}
				}
				// Command bindings
				cmdBindings := r.TwinEvent.GetTwinInterfaceCommandBindings(twinInterface, brokerExchange, twinInterfaceQueue)
				for _, cb := range cmdBindings {
					logger.Info(fmt.Sprintf("Creating Command Binding %s", cb.Name))
					err = r.Create(ctx, &cb, &client.CreateOptions{})
					if err != nil && !errors.IsAlreadyExists(err) {
						logger.Error(err, fmt.Sprintf("Error while creating command binding %s", cb.Name))
						resultErrors = append(resultErrors, err)
					}
				}
			}
		}
	}

	if len(resultErrors) > 0 {
		twinInterface.Status.Status = dtdv0.TwinInterfacePhaseFailed
		return ctrl.Result{}, resultErrors[0]
	}
	twinInterface.Status.Status = dtdv0.TwinInterfacePhaseRunning

	twinInterface.Labels = map[string]string{
		"twinscale/twin-interface": twinInterfaceName,
	}

	// Update Status for Running or Failed
	_, _ = r.updateTwinInterface(ctx, req, twinInterface)
	return ctrl.Result{}, nil
}

func (r *TwinInterfaceReconciler) getEventStoreQueue(ctx context.Context, twinInterface *dtdv0.TwinInterface) (rabbitmqv1beta1.Queue, error) {
	logger := log.FromContext(ctx)
	eventStoreQueuesList := rabbitmqv1beta1.QueueList{}
	queueListOptions := []client.ListOption{
		client.InNamespace(twinInterface.Namespace),
		client.MatchingLabels(client.MatchingFields{
			"eventing.knative.dev/trigger": "event-store-trigger",
		}),
	}

	err := r.List(ctx, &eventStoreQueuesList, queueListOptions...)
	if len(eventStoreQueuesList.Items) == 0 {
		logger.Error(err, fmt.Sprintf("No Queue found for event store %s", twinInterface.Name))
		return rabbitmqv1beta1.Queue{}, err
	}
	return eventStoreQueuesList.Items[0], nil
}

func (r *TwinInterfaceReconciler) getBrokerExchange(ctx context.Context, req ctrl.Request, twinInterface *dtdv0.TwinInterface) (rabbitmqv1beta1.Exchange, error) {
	logger := log.FromContext(ctx)
	exchangeList := rabbitmqv1beta1.ExchangeList{}
	exchangeListOptions := []client.ListOption{
		client.InNamespace(twinInterface.Namespace),
		client.MatchingLabels(client.MatchingFields{
			"eventing.knative.dev/broker": "twinscale",
		}),
	}

	err := r.List(ctx, &exchangeList, exchangeListOptions...)
	if err != nil {
		logger.Error(err, "Error while getting default broker exchange")
		return rabbitmqv1beta1.Exchange{}, err
	}
	return exchangeList.Items[0], nil
}

func (r *TwinInterfaceReconciler) getTwinInterfaceQueue(ctx context.Context, req ctrl.Request, twinInterface *dtdv0.TwinInterface) (rabbitmqv1beta1.Queue, error) {
	queueList := rabbitmqv1beta1.QueueList{}
	queueListOptions := []client.ListOption{
		client.InNamespace(twinInterface.Namespace),
		client.MatchingLabels(client.MatchingFields{
			"eventing.knative.dev/broker":  "twinscale",
			"eventing.knative.dev/trigger": twinInterface.Name,
		}),
	}

	err := r.List(ctx, &queueList, queueListOptions...)
	if err != nil {
		return rabbitmqv1beta1.Queue{}, err
	}
	if len(queueList.Items) == 0 {
		return rabbitmqv1beta1.Queue{}, errors.NewNotFound(rabbitmqv1beta1.Resource("rabbitmqv1beta1.Queue"), twinInterface.Name)
	}
	return queueList.Items[0], nil
}

func (r *TwinInterfaceReconciler) updateTwinInterface(ctx context.Context, req ctrl.Request, twinInterface *dtdv0.TwinInterface) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	err := r.Update(ctx, twinInterface, &client.UpdateOptions{})
	if err != nil {
		logger.Error(err, fmt.Sprintf("Error while updating TwinInterface %s", twinInterface.ObjectMeta.Name))
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TwinInterfaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// ensure helpers are initialized in long‐running manager
	if r.TwinService == nil {
		r.TwinService = twinservice.NewTwinService()
	}
	if r.TwinEvent == nil {
		r.TwinEvent = twinevent.NewTwinEvent()
	}
	if r.EventStore == nil {
		r.EventStore = eventStore.NewEventStore()
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&dtdv0.TwinInterface{}).
		Complete(r)
}
