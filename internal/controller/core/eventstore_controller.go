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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	corev0 "github.com/emereshub/twinscale-operator/api/core/v0"
	espkg "github.com/emereshub/twinscale-operator/pkg/event-store"
	keventing "knative.dev/eventing/pkg/apis/eventing/v1"
	kserving "knative.dev/serving/pkg/apis/serving/v1"
)

// EventStoreReconciler reconciles a EventStore object
//+kubebuilder:rbac:groups=core.core.twinscale.io,resources=eventstores,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.core.twinscale.io,resources=eventstores/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.core.twinscale.io,resources=eventstores/finalizers,verbs=update

type EventStoreReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	EventStore espkg.EventStore
}

// Reconcile is part of the main Kubernetes reconciliation loop.
func (r *EventStoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Initialize EventStore implementation if not set
	if r.EventStore == nil {
		r.EventStore = espkg.NewEventStore()
	}

	// Fetch the EventStore resource
	es := &corev0.EventStore{}
	err := r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, es)
	if err != nil {
		if errors.IsNotFound(err) {
			// Nothing to do
			return ctrl.Result{}, nil
		}
		logger.Error(err, fmt.Sprintf("Unexpected error fetching EventStore %s", req.Name))
		return ctrl.Result{}, err
	}

	// Reconcile underlying resources
	return r.createOrUpdateEventStoreResources(ctx, es)
}

// createOrUpdateEventStoreResources ensures the Knative Service and Trigger are present and up-to-date.
func (r *EventStoreReconciler) createOrUpdateEventStoreResources(ctx context.Context, es *corev0.EventStore) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// 1) Knative Service
	newSvc := r.EventStore.GetEventStoreService(es)
	err := r.Create(ctx, newSvc, &client.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		logger.Error(err, fmt.Sprintf("Error creating EventStore service %s", es.Name))
		return ctrl.Result{}, err
	} else if err != nil {
		// Service exists, update
		current := &kserving.Service{}
		if getErr := r.Get(ctx, types.NamespacedName{Namespace: es.Namespace, Name: es.Name}, current); getErr != nil {
			logger.Error(getErr, fmt.Sprintf("Error fetching existing Service %s", es.Name))
			return ctrl.Result{}, getErr
		}
		merged := r.EventStore.MergeEventStoreService(current, newSvc)
		if updateErr := r.Update(ctx, merged, &client.UpdateOptions{}); updateErr != nil {
			logger.Error(updateErr, fmt.Sprintf("Error updating EventStore service %s", es.Name))
			return ctrl.Result{}, updateErr
		}
		logger.Info(fmt.Sprintf("Event Store service %s updated", es.Name))
	} else {
		logger.Info(fmt.Sprintf("Event Store service %s created", es.Name))
	}

	// 2) Knative Trigger
	newTrig := r.EventStore.GetEventStoreTrigger(es)
	err = r.Create(ctx, newTrig, &client.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		logger.Error(err, fmt.Sprintf("Error creating trigger for EventStore %s", es.Name))
		return ctrl.Result{}, err
	} else if err != nil {
		// Trigger exists, update
		currentTrig := &keventing.Trigger{}
		triggerName := fmt.Sprintf("%s-trigger", es.Name)
		if getErr := r.Get(ctx, types.NamespacedName{Namespace: es.Namespace, Name: triggerName}, currentTrig); getErr != nil {
			logger.Error(getErr, fmt.Sprintf("Error fetching existing Trigger %s", triggerName))
			return ctrl.Result{}, getErr
		}
		mergedTrig := r.EventStore.MergeEventStoreTrigger(currentTrig, newTrig)
		if updateErr := r.Update(ctx, mergedTrig, &client.UpdateOptions{}); updateErr != nil {
			logger.Error(updateErr, fmt.Sprintf("Error updating trigger for EventStore %s", es.Name))
			return ctrl.Result{}, updateErr
		}
		logger.Info(fmt.Sprintf("Event Store trigger %s updated", triggerName))
	} else {
		logger.Info(fmt.Sprintf("Event Store trigger %s created", fmt.Sprintf("%s-trigger", es.Name)))
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EventStoreReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.EventStore == nil {
		r.EventStore = espkg.NewEventStore()
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev0.EventStore{}).
		Complete(r)
}
