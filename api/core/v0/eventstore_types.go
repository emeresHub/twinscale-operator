// api/core/v0/eventstore_types.go
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
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    corev1 "k8s.io/api/core/v1"
)

// AutoScalerType defines the type of autoscaler to use.
type AutoScalerType string

const (
    CONCURRENCY AutoScalerType = "concurrency"
    RPS         AutoScalerType = "rps"
    CPU         AutoScalerType = "cpu"
    MEMORY      AutoScalerType = "memory"
)

// EventStoreSpec defines the desired state of EventStore.
type EventStoreSpec struct {
    AutoScaling         EventStoreAutoScaling       `json:"autoScaling,omitempty"`
    Resources           corev1.ResourceRequirements `json:"resources,omitempty"`
    DispatcherResources corev1.ResourceRequirements `json:"dispatcherResources,omitempty"`
    Timeout             *int                        `json:"timeout,omitempty"`

    // Postgres holds connection info for a PostgreSQL-backed event store.
    Postgres PostgresConfig `json:"postgres"`
}

// PostgresConfig carries all details needed to connect to PostgreSQL.
type PostgresConfig struct {
    // Hostname or Service DNS of the Postgres server.
    Host string `json:"host"`
    // TCP port (default 5432).
    Port int `json:"port"`
    // Username to connect as.
    User string `json:"user"`
    // Database name.
    Database string `json:"database"`
    // Name of a Secret containing at least the "password" key.
    SecretName string `json:"secretName"`
}

// EventStoreAutoScaling defines auto-scaling behavior for the EventStore.
type EventStoreAutoScaling struct {
    MinScale                    *int           `json:"minScale,omitempty"`
    MaxScale                    *int           `json:"maxScale,omitempty"`
    Target                      *int           `json:"target,omitempty"`
    TargetUtilizationPercentage *int           `json:"targetUtilizationPercentage,omitempty"`
    Parallelism                 *int           `json:"parallelism,omitempty"`
    Metric                      AutoScalerType `json:"metric,omitempty"`
}

// EventStoreStatus defines the observed state of EventStore.
type EventStoreStatus struct {
    // INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
    // Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// EventStore is the Schema for the eventstores API.
type EventStore struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   EventStoreSpec   `json:"spec,omitempty"`
    Status EventStoreStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EventStoreList contains a list of EventStore.
type EventStoreList struct {
    metav1.TypeMeta `json:",inline"`
    metav1.ListMeta `json:"metadata,omitempty"`
    Items           []EventStore `json:"items"`
}

func init() {
    SchemeBuilder.Register(&EventStore{}, &EventStoreList{})
}
