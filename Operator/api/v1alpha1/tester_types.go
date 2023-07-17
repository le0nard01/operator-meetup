/*
Copyright 2023.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TesterSpec define a estrutura do objeto Tester, configurações dos testes
// são realizadas dentro do spec.
type TesterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster

	// Important: Run "make" to regenerate code after modifying this file
	// +required
	Tests *TestsStruct `json:"tests"`

	IgnoreControlPlane bool `json:"ignoreControlPlane,omitempty"`
}

// Os tipos dos testes que serão realizados
type TestsStruct struct {
	IopsTest *IopsTestStruct `json:"iopsTest,omitempty"`
}

// Teste de IOps utilizando a ferramenta fio, define os campos de limite
// de leitura e escrita, caso esses limites não sejam atingidos o node entrará
// em modo de Unschedulable.
type IopsTestStruct struct {
	Enabled        bool `json:"enabled,omitempty"`
	ReadThreshold  int  `json:"readThreshold,omitempty"`
	WriteThreshold int  `json:"writeThreshold,omitempty"`
}

// TesterStatus define o estado do objeto
type TesterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	NodesTested []string `json:"nodesTested"`
}

//+kubebuilder:object:root=true

// Tester is the Schema for the testers API
// +kubebuilder:subresource:status
type Tester struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TesterSpec   `json:"spec,omitempty"`
	Status TesterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TesterList contains a list of Tester
type TesterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Tester `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Tester{}, &TesterList{})
}
