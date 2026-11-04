package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
)

// SiteParameters are the configurable fields of a Matomo site (SitesManager API).
type SiteParameters struct {
	// SiteName is the display name of the website.
	// +kubebuilder:validation:Required
	SiteName string `json:"siteName"`

	// URLs are the URLs under which the site is reachable. The first is the main URL.
	// +kubebuilder:validation:MinItems=1
	URLs []string `json:"urls"`

	// Ecommerce enables ecommerce tracking for the site.
	// +optional
	Ecommerce *bool `json:"ecommerce,omitempty"`

	// SiteSearch enables site search tracking.
	// +optional
	SiteSearch *bool `json:"siteSearch,omitempty"`

	// Timezone is the site timezone (e.g. "Europe/Berlin").
	// +optional
	Timezone string `json:"timezone,omitempty"`

	// Currency is the ISO currency code used for ecommerce (e.g. "EUR").
	// +optional
	Currency string `json:"currency,omitempty"`

	// ExcludedIPs is a list of IPs excluded from tracking.
	// +optional
	ExcludedIPs []string `json:"excludedIps,omitempty"`

	// ExcludedQueryParameters is a list of URL query parameters stripped from tracked URLs.
	// +optional
	ExcludedQueryParameters []string `json:"excludedQueryParameters,omitempty"`

	// Type is the measurable type ("website" or "mobileapp").
	// +optional
	// +kubebuilder:validation:Enum=website;mobileapp
	Type string `json:"type,omitempty"`
}

// SiteObservation are the observable fields of a Site.
type SiteObservation struct {
	// IDSite is the Matomo numeric site ID.
	IDSite int `json:"idSite,omitempty"`

	// MainURL is the resolved main URL recorded in Matomo.
	MainURL string `json:"mainUrl,omitempty"`

	// TsCreated is the creation timestamp reported by Matomo.
	TsCreated string `json:"tsCreated,omitempty"`
}

// SiteSpec defines the desired state of a Site.
type SiteSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       SiteParameters `json:"forProvider"`
}

// SiteStatus represents the observed state of a Site.
type SiteStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          SiteObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,matomo}
// +kubebuilder:printcolumn:name="SYNCED",type=string,JSONPath=`.status.conditions[?(@.type=='Synced')].status`
// +kubebuilder:printcolumn:name="READY",type=string,JSONPath=`.status.conditions[?(@.type=='Ready')].status`
// +kubebuilder:printcolumn:name="ID",type=integer,JSONPath=`.status.atProvider.idSite`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

// A Site is a managed resource representing a Matomo tracked website.
type Site struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SiteSpec   `json:"spec"`
	Status SiteStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SiteList contains a list of Site.
type SiteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Site `json:"items"`
}
