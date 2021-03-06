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

// ProblemSpec defines the desired state of Problem
type ProblemSpec struct {
	Entities []ProblemEntity `json:"entities"`
}

type ProblemEntity struct {
	ID          string       `json:"id"`
	Constraints []Constraint `json:"constraints,omitempty"`
}

type Constraint struct {
	Mandatory      *bool     `json:"mandatory,omitempty"`
	Prohibited     *bool     `json:"prohibited,omitempty"`
	AtMostOf       *AtMostOf `json:"atMostOf,omitempty"`
	DependsOnOneOf []string  `json:"dependsOnOneOf,omitempty"`
	ConflictsWith  *string   `json:"conflictsWith,omitempty"`
}

type AtMostOf struct {
	Number int      `json:"number"`
	IDs    []string `json:"ids"`
}

// ProblemStatus defines the observed state of Problem
type ProblemStatus struct {
	Solution           []string `json:"solution,omitempty"`
	ObservedGeneration int64    `json:"observedGeneration,omitempty"`
	Error              string   `json:"error,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Problem is the Schema for the problems API
type Problem struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProblemSpec   `json:"spec,omitempty"`
	Status ProblemStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ProblemList contains a list of Problem
type ProblemList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Problem `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Problem{}, &ProblemList{})
}
