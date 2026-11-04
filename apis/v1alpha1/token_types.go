package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
)

// TokenParameters are the configurable fields of a Matomo app-specific
// auth token (UsersManager.createAppSpecificTokenAuth).
//
// Matomo returns the generated token only once, at creation, and stores it
// hashed. The token value is therefore written to the connection secret and
// cannot be read back or updated in place: changing any field forces a
// delete-and-recreate.
type TokenParameters struct {
	// UserLogin is the Matomo user the token is minted for.
	// +kubebuilder:validation:Required
	UserLogin string `json:"userLogin"`

	// PasswordConfirmationSecretRef references the user's password, which
	// Matomo requires to confirm token creation.
	PasswordConfirmationSecretRef xpv1.SecretKeySelector `json:"passwordConfirmationSecretRef"`

	// Description is the human-readable token description. Used as the
	// external name for correlation since Matomo exposes no stable token ID.
	// +kubebuilder:validation:Required
	Description string `json:"description"`

	// ExpireHours sets a relative expiry, in hours, from creation time.
	// +optional
	ExpireHours *int64 `json:"expireHours,omitempty"`
}

// TokenObservation are the observable fields of a Token.
type TokenObservation struct {
	// CreatedAt records when the provider minted the token.
	CreatedAt string `json:"createdAt,omitempty"`
}

// TokenSpec defines the desired state of a Token.
type TokenSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       TokenParameters `json:"forProvider"`
}

// TokenStatus represents the observed state of a Token.
type TokenStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          TokenObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,matomo}
// +kubebuilder:printcolumn:name="SYNCED",type=string,JSONPath=`.status.conditions[?(@.type=='Synced')].status`
// +kubebuilder:printcolumn:name="READY",type=string,JSONPath=`.status.conditions[?(@.type=='Ready')].status`
// +kubebuilder:printcolumn:name="USER",type=string,JSONPath=`.spec.forProvider.userLogin`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

// A Token is a managed resource representing a Matomo app-specific auth token.
// The minted token is written to the resource's connection secret.
type Token struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TokenSpec   `json:"spec"`
	Status TokenStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TokenList contains a list of Token.
type TokenList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Token `json:"items"`
}
