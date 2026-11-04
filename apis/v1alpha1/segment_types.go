package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
)

// SegmentParameters are the configurable fields of a Matomo segment (SegmentEditor API).
type SegmentParameters struct {
	// Name is the display name of the segment.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Definition is the Matomo segment definition string
	// (e.g. "visitCountry==de;visitorType==returning").
	// +kubebuilder:validation:Required
	Definition string `json:"definition"`

	// IDSite restricts the segment to a single site ID. Omit for all sites.
	// +optional
	IDSite *int `json:"idSite,omitempty"`

	// AutoArchive pre-processes the segment during archiving instead of on demand.
	// +optional
	AutoArchive *bool `json:"autoArchive,omitempty"`

	// EnabledAllUsers makes the segment visible to all users.
	// +optional
	EnabledAllUsers *bool `json:"enabledAllUsers,omitempty"`
}

// SegmentObservation are the observable fields of a Segment.
type SegmentObservation struct {
	// IDSegment is the Matomo numeric segment ID.
	IDSegment int `json:"idSegment,omitempty"`

	// Definition is the definition currently stored in Matomo.
	Definition string `json:"definition,omitempty"`
}

// SegmentSpec defines the desired state of a Segment.
type SegmentSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       SegmentParameters `json:"forProvider"`
}

// SegmentStatus represents the observed state of a Segment.
type SegmentStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          SegmentObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,matomo}
// +kubebuilder:printcolumn:name="SYNCED",type=string,JSONPath=`.status.conditions[?(@.type=='Synced')].status`
// +kubebuilder:printcolumn:name="READY",type=string,JSONPath=`.status.conditions[?(@.type=='Ready')].status`
// +kubebuilder:printcolumn:name="ID",type=integer,JSONPath=`.status.atProvider.idSegment`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

// A Segment is a managed resource representing a Matomo saved segment.
type Segment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SegmentSpec   `json:"spec"`
	Status SegmentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SegmentList contains a list of Segment.
type SegmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Segment `json:"items"`
}
