package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// LocalUploadProxySpec defines the desired state of LocalUploadProxy
type LocalUploadProxySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	NodeName string                           `json:"nodeName"`
	PVC      corev1.PersistentVolumeClaimSpec `json:"pvc"`
}

// LocalUploadProxyState is the current state of LocalUploadProxy
type LocalUploadProxyState string

const (
	// LocalUploadProxyStateCreating indicates LocalUploadProxy is creating
	LocalUploadProxyStateCreating LocalUploadProxyState = "Creating"
	// LocalUploadProxyStateAvailable indicates LocalUploadProxy is available
	LocalUploadProxyStateAvailable LocalUploadProxyState = "Available"
	// LocalUploadProxyStateError indicates LocalUploadProxy is error
	LocalUploadProxyStateError LocalUploadProxyState = "Error"
)

// LocalUploadProxyStatus defines the observed state of LocalUploadProxy
type LocalUploadProxyStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	// State is the current state of LocalUploadProxy
	State LocalUploadProxyState `json:"state"`
	// Conditions indicate current conditions of LocalUploadProxy
	// +optional
	Conditions []Condition `json:"conditions,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LocalUploadProxy is the Schema for the localuploadproxies API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=localuploadproxies,scope=Namespaced
type LocalUploadProxy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LocalUploadProxySpec   `json:"spec,omitempty"`
	Status LocalUploadProxyStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LocalUploadProxyList contains a list of LocalUploadProxy
type LocalUploadProxyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LocalUploadProxy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LocalUploadProxy{}, &LocalUploadProxyList{})
}
