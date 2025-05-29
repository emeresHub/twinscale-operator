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

// HistoricalDataStoreSpec defines the desired state of HistoricalDataStore
type HistoricalDataStoreSpec struct {
	AutoScaling         HistoricalDataStoreAutoScaling `json:"autoScaling,omitempty"`
	Resources           corev1.ResourceRequirements   `json:"resources,omitempty"`
	DispatcherResources corev1.ResourceRequirements   `json:"dispatcherResources,omitempty"`
	Timeout             *int                            `json:"timeout,omitempty"`
	
	// Postgres holds connection info for a PostgreSQL-backed historical store.
	Postgres PostgresConfig `json:"postgres"`
}


type HistoricalDataStoreAutoScaling struct {
	MinScale                    *int `json:"minScale,omitempty"`
	MaxScale                    *int `json:"maxScale,omitempty"`
	Target                      *int `json:"target,omitempty"`
	TargetUtilizationPercentage *int `json:"targetUtilizationPercentage,omitempty"`
	Parallelism                 *int `json:"parallelism,omitempty"`
	// KNative Metric values (default, if not informed: concurrency)
	// concurrency: the number of simultaneous requests that can be processed by each replica of an application at any given time
	// rps: requests per seconds
	// cpu: cpu usage
	// memory: memory usage
	Metric AutoScalerType `json:"metric,omitempty"`
}

// HistoricalDataStoreStatus defines the observed state of HistoricalDataStore
type HistoricalDataStoreStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// HistoricalDataStore is the Schema for the historicaldatastores API
type HistoricalDataStore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HistoricalDataStoreSpec   `json:"spec,omitempty"`
	Status HistoricalDataStoreStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// HistoricalDataStoreList contains a list of HistoricalDataStore
type HistoricalDataStoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HistoricalDataStore `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HistoricalDataStore{}, &HistoricalDataStoreList{})
}
