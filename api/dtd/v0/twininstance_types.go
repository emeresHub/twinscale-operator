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

type TwinInstancePhase string

const (
	TwinInstancePhasePending TwinInstancePhase = "Pending"
	TwinInstancePhaseUnknown TwinInstancePhase = "Unknown"
	TwinInstancePhaseRunning TwinInstancePhase = "Running"
	TwinInstancePhaseFailed  TwinInstancePhase = "Failed"
)

// TwinInstanceSpec defines the desired state of TwinInstance
type TwinInstanceSpec struct {
	Interface                 string                        `json:"interface,omitempty"`
	EndpointSettings          *TwinInstanceEndpointSettings `json:"endpointSettings,omitempty"`
	Data                      *TwinInstanceDataSpec         `json:"data,omitempty"`
	TwinInstanceRelationships []TwinInstanceRelationship    `json:"twinInstanceRelationships,omitempty"`
}

type TwinInstanceDataSpec struct {
	Properties  []TwinInstancePropertyData  `json:"properties,omitempty"`  // TODO: read-only
	Telemetries []TwinInstanceTelemetryData `json:"telemetries,omitempty"` // TODO: read-only
}

// TODO: read-only
type TwinInstancePropertyData struct {
	Id    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Value string `json:"value"`
}

// TODO: read-only
type TwinInstanceTelemetryData struct {
	Id    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Value string `json:"value"`
}

type TwinInstanceRelationship struct {
	// The TwinInstance Relationship name
	Name string `json:"name"`
	// The Target TwinInterface of the Relationship
	Interface string `json:"interface"`
	// The Target TwinInstance of the Relationship
	Instance string `json:"instance"`
}

// TODO: Configure as read-only
type TwinInstanceEndpointSettings struct {
	HttpEndpoint *TwinInstanceHttpEndpointSettings `json:"httpEndpoint,omitempty"`
	MqttEndpoint *TwinInstanceMqttEndpointSettings `json:"mqttEndpoint,omitempty"`
	AmqpEndpoint *TwinInstanceAmqpEndpointSettings `json:"amqpEndpoint,omitempty"`
}

// TODO: Configure as read-only
type TwinInstanceHttpEndpointSettings struct {
	Url string `json:"url,omitempty"`
}

// TODO: Configure as read-only
type TwinInstanceMqttEndpointSettings struct {
	Url             string `json:"url,omitempty"`
	PublisherTopic  string `json:"publisherTopic,omitempty"`
	SubscriberTopic string `json:"subscriberTopic,omitempty"`
}

type TwinInstanceAmqpEndpointSettings struct {
	Url             string `json:"url,omitempty"`
	PublisherTopic  string `json:"publisherTopic,omitempty"`
	SubscriberTopic string `json:"subscriberTopic,omitempty"`
}

type TwinInterfaceEventsSink struct {
	InstanceId string `json:"instanceId,omitempty"`
}

// TwinInstanceStatus defines the observed state of TwinInstance
type TwinInstanceStatus struct {
	Status TwinInstancePhase `json:"status,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// TwinInstance is the Schema for the twininstances API
type TwinInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TwinInstanceSpec   `json:"spec,omitempty"`
	Status TwinInstanceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TwinInstanceList contains a list of TwinInstance
type TwinInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TwinInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TwinInstance{}, &TwinInstanceList{})
}
