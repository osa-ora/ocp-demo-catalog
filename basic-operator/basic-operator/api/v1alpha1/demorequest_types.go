/*
Copyright 2026.

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DemoRequestSpec defines the desired state of DemoRequest.
type DemoRequestSpec struct {
	DemoName  string `json:"demoName"`
	Namespace string `json:"namespace,omitempty"`
}

// DemoRequestStatus defines the observed state of DemoRequest.
type DemoRequestStatus struct {
	Phase   string `json:"phase,omitempty"`
	Message string `json:"message,omitempty"`

	Resources []corev1.ObjectReference `json:"resources,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// DemoRequest is the Schema for the demorequests API.
type DemoRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DemoRequestSpec   `json:"spec,omitempty"`
	Status DemoRequestStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DemoRequestList contains a list of DemoRequest.
type DemoRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DemoRequest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DemoRequest{}, &DemoRequestList{})
}
