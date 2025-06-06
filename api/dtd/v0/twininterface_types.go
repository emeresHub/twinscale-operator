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

type TwinInterfacePhase string

const (
	TwinInterfacePhasePending TwinInterfacePhase = "Pending"
	TwinInterfacePhaseUnknown TwinInterfacePhase = "Unknown"
	TwinInterfacePhaseRunning TwinInterfacePhase = "Running"
	TwinInterfacePhaseFailed  TwinInterfacePhase = "Failed"
)

type PrimitiveType string
type ComplexType string
type Multiplicity string
type AutoScalerType string

const (
	Integer PrimitiveType = "integer"
	String  PrimitiveType = "string"
	Boolean PrimitiveType = "boolean"
	Double  PrimitiveType = "double"
)

const (
	Object ComplexType = "Object"
)

const (
	ONE  Multiplicity = "one"
	MANY Multiplicity = "many"
)

const (
	CONCURRENCY AutoScalerType = "concurrency"
	RPS         AutoScalerType = "rps"
	CPU         AutoScalerType = "cpu"
	MEMORY      AutoScalerType = "memory"
)

// TwinInterfaceSpec defines the desired state of TwinInterface
type TwinInterfaceSpec struct {
	Id               string                  `json:"id,omitempty"`
	DisplayName      string                  `json:"displayName,omitempty"`
	Description      string                  `json:"description,omitempty"`
	Comment          string                  `json:"comment,omitempty"`
	Properties       []TwinProperty          `json:"properties,omitempty"`
	Commands         []TwinCommand           `json:"commands,omitempty"`
	Relationships    []TwinRelationship      `json:"relationships,omitempty"`
	Telemetries      []TwinTelemetry         `json:"telemetries,omitempty"`
	ExtendsInterface string                  `json:"extendsInterface,omitempty"`
	EventStore       TwinInterfaceEventStore `json:"eventStore,omitempty"`
	Service          *TwinInterfaceService   `json:"service,omitempty"` // Must be a pointer because Containers[] field is required
}

type TwinInterfaceService struct {
	Template    corev1.PodTemplateSpec   `json:"template,omitempty"`
	AutoScaling TwinInterfaceAutoScaling `json:"autoScaling,omitempty"`
}

// KNative Pod Auto Scaler Settings
type TwinInterfaceAutoScaling struct {
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

type TwinInterfaceEventStore struct {
	PersistRealEvent    bool `json:"persistRealEvent,omitempty"`
	PersistVirtualEvent bool `json:"persistVirtualEvent,omitempty"`
}

type TwinProperty struct {
	Id          string      `json:"id,omitempty"`
	Comment     string      `json:"comment,omitempty"`
	Description string      `json:"description,omitempty"`
	DisplayName string      `json:"displayName,omitempty"`
	Name        string      `json:"name,omitempty"`
	Schema      *TwinSchema `json:"schema,omitempty"`
	Writeable   bool        `json:"writable,omitempty"`
}

type TwinCommand struct {
	Id          string          `json:"id,omitempty"`
	Comment     string          `json:"comment,omitempty"`
	Description string          `json:"description,omitempty"`
	DisplayName string          `json:"displayName,omitempty"`
	Name        string          `json:"name,omitempty"`
	Request     CommandRequest  `json:"request,omitempty"`
	Response    CommandResponse `json:"response,omitempty"`
}

type CommandRequest struct {
	Name        string      `json:"name,omitempty"`
	DisplayName string      `json:"displayName,omitempty"`
	Description string      `json:"description,omitempty"`
	Schema      *TwinSchema `json:"schema,omitempty"`
}

type CommandResponse struct {
	Name        string      `json:"name,omitempty"`
	DisplayName string      `json:"displayName,omitempty"`
	Description string      `json:"description,omitempty"`
	Schema      *TwinSchema `json:"schema,omitempty"`
}

type TwinRelationship struct {
	Id              string         `json:"id,omitempty"`
	Comment         string         `json:"comment,omitempty"`
	Description     string         `json:"description,omitempty"`
	DisplayName     string         `json:"displayName,omitempty"`
	MaxMultiplicity int            `json:"maxMultiplicity,omitempty"`
	MinMultiplicity int            `json:"minMultiplicity,omitempty"`
	Name            string         `json:"name,omitempty"`
	Properties      []TwinProperty `json:"properties,omitempty"`
	Interface       string         `json:"interface,omitempty"`
	Schema          *TwinSchema    `json:"schema,omitempty"`
	Writeable       bool           `json:"writeable,omitempty"`
	// Indicate if the data must be aggregated in the relationship parent
	AggregateData bool `json:"aggregateData,omitempty"`
}

type TwinTelemetry struct {
	Id          string      `json:"id,omitempty"`
	Comment     string      `json:"comment,omitempty"`
	Description string      `json:"description,omitempty"`
	DisplayName string      `json:"displayName,omitempty"`
	Name        string      `json:"name,omitempty"`
	Schema      *TwinSchema `json:"schema,omitempty"`
}

type TwinSchema struct {
	PrimitiveType PrimitiveType    `json:"primitiveType,omitempty"`
	ComplexType   *TwinComplexType `json:"complexType,omitempty"`
	EnumType      *TwinEnumSchema  `json:"enumType,omitempty"`
}

type TwinComplexType struct {
	Type   ComplexType             `json:"type,omitempty"`
	Fields []TwinComplexTypeFields `json:"fields,omitempty"`
}

type TwinComplexTypeFields struct {
	Name   string                 `json:"name,omitempty"`
	Schema *TwinComplexTypeSchema `json:"schema,omitempty"`
}

type TwinComplexTypeSchema struct {
	PrimitiveType PrimitiveType `json:"primitiveType,omitempty"`
}

type TwinEnumSchema struct {
	ValueSchema PrimitiveType          `json:"valueSchema,omitempty"`
	EnumValues  []TwinEnumSchemaValues `json:"enumValues,omitempty"`
}

type TwinEnumSchemaValues struct {
	Name        string `json:"name,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	EnumValue   string `json:"enumValue,omitempty"`
}

type TwinObjectSchema struct {
	ValueSchema PrimitiveType          `json:"valueSchema,omitempty"`
	EnumValues  []TwinEnumSchemaValues `json:"enumValues,omitempty"`
}

type TwinObjectSchemaFields struct {
	Name        string `json:"name,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	Schema      string `json:"enumValue,omitempty"`
}

// TwinInterfaceStatus defines the observed state of TwinInterface
type TwinInterfaceStatus struct {
	Status TwinInterfacePhase `json:"status,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// TwinInterface is the Schema for the twininterfaces API
type TwinInterface struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TwinInterfaceSpec   `json:"spec,omitempty"`
	Status TwinInterfaceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TwinInterfaceList contains a list of TwinInterface
type TwinInterfaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TwinInterface `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TwinInterface{}, &TwinInterfaceList{})
}
