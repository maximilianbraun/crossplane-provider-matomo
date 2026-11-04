package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
)

// TLSConfig configures TLS for the Matomo connection.
type TLSConfig struct {
	// CASecretRef references a secret holding a PEM CA bundle to trust.
	// +optional
	CASecretRef *xpv1.SecretKeySelector `json:"caSecretRef,omitempty"`

	// InsecureSkipVerify disables TLS certificate verification. Do not use in production.
	// +optional
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty"`
}

// ProviderCredentials defines how to authenticate to the Matomo Reporting API.
type ProviderCredentials struct {
	// TokenAuthSecretRef references a secret key holding a Matomo token_auth
	// value with sufficient privileges (typically a superuser token).
	TokenAuthSecretRef xpv1.SecretKeySelector `json:"tokenAuthSecretRef"`
}

// ProviderConfigSpec defines the desired state of a ProviderConfig.
type ProviderConfigSpec struct {
	// URL is the base URL of the Matomo instance (e.g. https://analytics.example.com).
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^https?://`
	URL string `json:"url"`

	// Credentials authenticate the provider to Matomo.
	Credentials ProviderCredentials `json:"credentials"`

	// TLS configures the connection TLS behaviour.
	// +optional
	TLS *TLSConfig `json:"tls,omitempty"`
}

// ProviderConfigStatus represents the status of a ProviderConfig.
type ProviderConfigStatus struct {
	xpv1.ProviderConfigStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,provider,matomo}
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:printcolumn:name="URL",type=string,JSONPath=`.spec.url`

// A ProviderConfig configures how this provider connects to a Matomo instance.
type ProviderConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProviderConfigSpec   `json:"spec"`
	Status ProviderConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProviderConfigList contains a list of ProviderConfig.
type ProviderConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProviderConfig `json:"items"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,categories={crossplane,provider,matomo}

// A ProviderConfigUsage indicates that a resource is using a ProviderConfig.
type ProviderConfigUsage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	xpv1.ProviderConfigUsage `json:",inline"`
}

// +kubebuilder:object:root=true

// ProviderConfigUsageList contains a list of ProviderConfigUsage.
type ProviderConfigUsageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProviderConfigUsage `json:"items"`
}
