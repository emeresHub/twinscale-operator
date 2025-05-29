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

	corev1 "k8s.io/api/core/v1"
	corev0 "github.com/emereshub/twinscale-operator/api/core/v0"
	hdsPkg "github.com/emereshub/twinscale-operator/pkg/historical-data-store"
	keventing "knative.dev/eventing/pkg/apis/eventing/v1"
	kserving "knative.dev/serving/pkg/apis/serving/v1"
)

// HistoricalDataStoreReconciler reconciles a HistoricalDataStore object
//+kubebuilder:rbac:groups=core.core.twinscale.io,resources=historicaldatastores,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.core.twinscale.io,resources=historicaldatastores/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.core.twinscale.io,resources=historicaldatastores/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list

type HistoricalDataStoreReconciler struct {
	client.Client
	Scheme                *runtime.Scheme
	HistoricalDataStore   hdsPkg.HistoricalDataStore
}

// Reconcile is part of the main Kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *HistoricalDataStoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Ensure HistoricalDataStore implementation is initialized
	if r.HistoricalDataStore == nil {
		r.HistoricalDataStore = hdsPkg.NewHistoricalDataStore()
	}

	// 1) Fetch the HistoricalDataStore resource
	hds := &corev0.HistoricalDataStore{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: req.Name}, hds); err != nil {
		if errors.IsNotFound(err) {
			// CR was deleted—nothing to do
			return ctrl.Result{}, nil
		}
		logger.Error(err, fmt.Sprintf("Unexpected error fetching HistoricalDataStore %s", req.Name))
		return ctrl.Result{}, err
	}

	// 2) Verify Postgres Secret exists (fail fast if missing)
	if hds.Spec.Postgres.SecretName != "" {
		sec := &corev1.Secret{}
		if err := r.Get(ctx,
			types.NamespacedName{Namespace: hds.Namespace, Name: hds.Spec.Postgres.SecretName},
			sec,
		); err != nil {
			if errors.IsNotFound(err) {
				logger.Error(err, "Postgres Secret not found", "secret", hds.Spec.Postgres.SecretName)
			}
			return ctrl.Result{}, err
		}
	}

	// 3) Reconcile service and trigger
	return r.createOrUpdateHistoricalDataStoreResources(ctx, hds)
}

// createOrUpdateHistoricalDataStoreResources ensures the Knative Service and Trigger are present and up-to-date.
func (r *HistoricalDataStoreReconciler) createOrUpdateHistoricalDataStoreResources(ctx context.Context, hds *corev0.HistoricalDataStore) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// 1) Knative Service
	newSvc := r.HistoricalDataStore.GetService(hds)
	err := r.Create(ctx, newSvc, &client.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		logger.Error(err, fmt.Sprintf("Error creating HistoricalDataStore service %s", hds.Name))
		return ctrl.Result{}, err
	} else if err != nil {
		// Service exists—update
		current := &kserving.Service{}
		if getErr := r.Get(ctx, types.NamespacedName{Namespace: hds.Namespace, Name: hds.Name}, current); getErr != nil {
			logger.Error(getErr, fmt.Sprintf("Error fetching existing service %s", hds.Name))
			return ctrl.Result{}, getErr
		}
		merged := r.HistoricalDataStore.MergeService(current, newSvc)
		if updateErr := r.Update(ctx, merged, &client.UpdateOptions{}); updateErr != nil {
			logger.Error(updateErr, fmt.Sprintf("Error updating HistoricalDataStore service %s", hds.Name))
			return ctrl.Result{}, updateErr
		}
		logger.Info(fmt.Sprintf("HistoricalDataStore service %s updated", hds.Name))
	} else {
		logger.Info(fmt.Sprintf("HistoricalDataStore service %s created", hds.Name))
	}

	// 2) Knative Trigger
	newTrig := r.HistoricalDataStore.GetTrigger(hds)
	err = r.Create(ctx, newTrig, &client.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		logger.Error(err, fmt.Sprintf("Error creating trigger for HistoricalDataStore %s", hds.Name))
		return ctrl.Result{}, err
	} else if err != nil {
		// Trigger exists—update
		currentTrig := &keventing.Trigger{}
		triggerName := fmt.Sprintf("%s-trigger", hds.Name)
		if getErr := r.Get(ctx, types.NamespacedName{Namespace: hds.Namespace, Name: triggerName}, currentTrig); getErr != nil {
			logger.Error(getErr, fmt.Sprintf("Error fetching existing trigger %s", triggerName))
			return ctrl.Result{}, getErr
		}
		mergedTrig := r.HistoricalDataStore.MergeTrigger(currentTrig, newTrig)
		if updateErr := r.Update(ctx, mergedTrig, &client.UpdateOptions{}); updateErr != nil {
			logger.Error(updateErr, fmt.Sprintf("Error updating trigger for HistoricalDataStore %s", hds.Name))
			return ctrl.Result{}, updateErr
		}
		logger.Info(fmt.Sprintf("HistoricalDataStore trigger %s updated", triggerName))
	} else {
		logger.Info(fmt.Sprintf("HistoricalDataStore trigger %s created", fmt.Sprintf("%s-trigger", hds.Name)))
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HistoricalDataStoreReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.HistoricalDataStore == nil {
		r.HistoricalDataStore = hdsPkg.NewHistoricalDataStore()
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev0.HistoricalDataStore{}).
		Complete(r)
}
