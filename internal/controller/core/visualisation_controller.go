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
	visPkg "github.com/emereshub/twinscale-operator/pkg/visualisation"
	keventing "knative.dev/eventing/pkg/apis/eventing/v1"
	kserving "knative.dev/serving/pkg/apis/serving/v1"
)

// VisualisationReconciler reconciles a Visualisation object
//+kubebuilder:rbac:groups=core.core.twinscale.io,resources=visualisations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.core.twinscale.io,resources=visualisations/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.core.twinscale.io,resources=visualisations/finalizers,verbs=update

type VisualisationReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	Visualisation visPkg.Visualisation
}

// Reconcile is part of the main Kubernetes reconciliation loop.
func (r *VisualisationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Ensure Visualisation implementation is initialized
	if r.Visualisation == nil {
		r.Visualisation = visPkg.NewVisualisation()
	}

	// Fetch the Visualisation resource
	viz := &corev0.Visualisation{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: req.Name}, viz); err != nil {
		if errors.IsNotFound(err) {
			// Resource deleted—nothing to do
			return ctrl.Result{}, nil
		}
		logger.Error(err, fmt.Sprintf("Unexpected error fetching Visualisation %s", req.Name))
		return ctrl.Result{}, err
	}

	// Reconcile Knative Service and Trigger
	return r.createOrUpdateVisualisationResources(ctx, viz)
}

// createOrUpdateVisualisationResources ensures the Knative Service and Trigger are present and up-to-date.
func (r *VisualisationReconciler) createOrUpdateVisualisationResources(ctx context.Context, viz *corev0.Visualisation) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// 1) Knative Service
	newSvc := r.Visualisation.GetService(viz)
	err := r.Create(ctx, newSvc, &client.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		logger.Error(err, fmt.Sprintf("Error creating Visualisation service %s", viz.Name))
		return ctrl.Result{}, err
	} else if err != nil {
		// Service exists—update
		current := &kserving.Service{}
		if getErr := r.Get(ctx, types.NamespacedName{Namespace: viz.Namespace, Name: viz.Name}, current); getErr != nil {
			logger.Error(getErr, fmt.Sprintf("Error fetching existing service %s", viz.Name))
			return ctrl.Result{}, getErr
		}
		merged := r.Visualisation.MergeService(current, newSvc)
		if updateErr := r.Update(ctx, merged, &client.UpdateOptions{}); updateErr != nil {
			logger.Error(updateErr, fmt.Sprintf("Error updating Visualisation service %s", viz.Name))
			return ctrl.Result{}, updateErr
		}
		logger.Info(fmt.Sprintf("Visualisation service %s updated", viz.Name))
	} else {
		logger.Info(fmt.Sprintf("Visualisation service %s created", viz.Name))
	}

	// 2) Knative Trigger
	newTrig := r.Visualisation.GetTrigger(viz)
	err = r.Create(ctx, newTrig, &client.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		logger.Error(err, fmt.Sprintf("Error creating trigger for Visualisation %s", viz.Name))
		return ctrl.Result{}, err
	} else if err != nil {
		// Trigger exists—update
		currentTrig := &keventing.Trigger{}
		triggerName := fmt.Sprintf("%s-trigger", viz.Name)
		if getErr := r.Get(ctx, types.NamespacedName{Namespace: viz.Namespace, Name: triggerName}, currentTrig); getErr != nil {
			logger.Error(getErr, fmt.Sprintf("Error fetching existing trigger %s", triggerName))
			return ctrl.Result{}, getErr
		}
		mergedTrig := r.Visualisation.MergeTrigger(currentTrig, newTrig)
		if updateErr := r.Update(ctx, mergedTrig, &client.UpdateOptions{}); updateErr != nil {
			logger.Error(updateErr, fmt.Sprintf("Error updating trigger for Visualisation %s", viz.Name))
			return ctrl.Result{}, updateErr
		}
		logger.Info(fmt.Sprintf("Visualisation trigger %s updated", triggerName))
	} else {
		logger.Info(fmt.Sprintf("Visualisation trigger %s created", fmt.Sprintf("%s-trigger", viz.Name)))
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VisualisationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.Visualisation == nil {
		r.Visualisation = visPkg.NewVisualisation()
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev0.Visualisation{}).
		Complete(r)
}
