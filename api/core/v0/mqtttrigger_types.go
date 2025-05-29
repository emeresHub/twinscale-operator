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
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MQTTTriggerSpec defines the desired state of MQTTTrigger
type MQTTTriggerSpec struct {
    // INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
    // Important: Run "make" to regenerate code after modifying this file

    // Foo is an example field of MQTTTrigger. Edit mqtttrigger_types.go to remove/update
    Foo string `json:"foo,omitempty"`
}

// MQTTTriggerStatus defines the observed state of MQTTTrigger
type MQTTTriggerStatus struct {
    // Phase is the current lifecycle phase of this MQTTTrigger (e.g. "Ready", "Error").
    Phase string `json:"phase,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// MQTTTrigger is the Schema for the mqtttriggers API
type MQTTTrigger struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   MQTTTriggerSpec   `json:"spec,omitempty"`
    Status MQTTTriggerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MQTTTriggerList contains a list of MQTTTrigger
type MQTTTriggerList struct {
    metav1.TypeMeta `json:",inline"`
    metav1.ListMeta `json:"metadata,omitempty"`
    Items           []MQTTTrigger `json:"items"`
}

func init() {
    SchemeBuilder.Register(&MQTTTrigger{}, &MQTTTriggerList{})
}
