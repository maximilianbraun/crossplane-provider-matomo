package v1alpha1

import (
	"reflect"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

const (
	// Group is the API group for all Matomo resources.
	Group = "matomo.m.crossplane.io"
	// Version is the API version.
	Version = "v1alpha1"
)

var (
	// SchemeGroupVersion is the group/version for this API.
	SchemeGroupVersion = schema.GroupVersion{Group: Group, Version: Version}
	// SchemeBuilder registers the types with a scheme.
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion} //nolint:staticcheck // standard Crossplane pattern
	// AddToScheme adds all registered types to a scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

func init() {
	SchemeBuilder.Register(
		&ProviderConfig{}, &ProviderConfigList{},
		&ProviderConfigUsage{}, &ProviderConfigUsageList{},
		&Site{}, &SiteList{},
		&Segment{}, &SegmentList{},
		&Token{}, &TokenList{},
	)
}

// Site type metadata.
var (
	SiteKind             = reflect.TypeFor[Site]().Name()
	SiteGroupKind        = schema.GroupKind{Group: Group, Kind: SiteKind}.String()
	SiteGroupVersionKind = SchemeGroupVersion.WithKind(SiteKind)
)

// Segment type metadata.
var (
	SegmentKind             = reflect.TypeFor[Segment]().Name()
	SegmentGroupKind        = schema.GroupKind{Group: Group, Kind: SegmentKind}.String()
	SegmentGroupVersionKind = SchemeGroupVersion.WithKind(SegmentKind)
)

// Token type metadata.
var (
	TokenKind             = reflect.TypeFor[Token]().Name()
	TokenGroupKind        = schema.GroupKind{Group: Group, Kind: TokenKind}.String()
	TokenGroupVersionKind = SchemeGroupVersion.WithKind(TokenKind)
)
