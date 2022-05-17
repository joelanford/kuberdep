/*
Copyright 2022.

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

package v1

import (
	"encoding/json"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// EntitySpec defines the desired state of Entity
type EntitySpec struct {
	Object      runtime.RawExtension `json:"object,omitempty"`
	Properties  []EntityProperty     `json:"properties,omitempty"`
	Constraints []EntityConstraint   `json:"constraints,omitempty"`
}

type EntityProperty struct {
	Type string `json:"type"`

	//+kubebuilder:validation:Schemaless
	//+kubebuilder:validation:XPreserveUnknownFields
	Value json.RawMessage `json:"value,omitempty"`
}

type EntityConstraint struct {
	Type string `json:"type"`

	//+kubebuilder:validation:Schemaless
	//+kubebuilder:validation:XPreserveUnknownFields
	Value json.RawMessage `json:"value,omitempty"`
}

func ValueToInterface(v json.RawMessage) (interface{}, error) {
	if len(v) == 0 {
		return nil, nil
	}
	var i interface{}
	if err := json.Unmarshal(v, &i); err != nil {
		return nil, err
	}
	return i, nil
}

// EntityStatus defines the observed state of Entity
type EntityStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Entity is the Schema for the entities API
type Entity struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EntitySpec   `json:"spec,omitempty"`
	Status EntityStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EntityList contains a list of Entity
type EntityList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Entity `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Entity{}, &EntityList{})
}
