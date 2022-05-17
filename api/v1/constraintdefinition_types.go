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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ConstraintDefinitionSpec defines the desired state of ConstraintDefinition
type ConstraintDefinitionSpec struct {
	ID   string `json:"id"`
	Body string `json:"body"`
}

// ConstraintDefinitionStatus defines the observed state of ConstraintDefinition
type ConstraintDefinitionStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ConstraintDefinition is the Schema for the constraintdefinitions API
type ConstraintDefinition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConstraintDefinitionSpec   `json:"spec,omitempty"`
	Status ConstraintDefinitionStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ConstraintDefinitionList contains a list of ConstraintDefinition
type ConstraintDefinitionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ConstraintDefinition `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ConstraintDefinition{}, &ConstraintDefinitionList{})
}
