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

package v0

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VisualisationSpec defines the desired state of Visualisation.
type VisualisationSpec struct {
	AutoScaling         VisualisationAutoScaling    `json:"autoScaling,omitempty"`
	Resources           corev1.ResourceRequirements `json:"resources,omitempty"`
	DispatcherResources corev1.ResourceRequirements `json:"dispatcherResources,omitempty"`
	Timeout             *int                        `json:"timeout,omitempty"`
}

type VisualisationAutoScaling struct {
	MinScale                    *int           `json:"minScale,omitempty"`
	MaxScale                    *int           `json:"maxScale,omitempty"`
	Target                      *int           `json:"target,omitempty"`
	TargetUtilizationPercentage *int           `json:"targetUtilizationPercentage,omitempty"`
	Parallelism                 *int           `json:"parallelism,omitempty"`
	Metric                      AutoScalerType `json:"metric,omitempty"`
}

// VisualisationStatus defines the observed state of Visualisation.
type VisualisationStatus struct {
	Status     string   `json:"status,omitempty"`
	Conditions []string `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Visualisation is the Schema for the visualisations API.
type Visualisation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VisualisationSpec   `json:"spec,omitempty"`
	Status VisualisationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VisualisationList contains a list of Visualisation.
type VisualisationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Visualisation `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Visualisation{}, &VisualisationList{})
}
