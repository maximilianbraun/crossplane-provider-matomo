package segment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	xpresource "github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/maximilianbraun/provider-matomo/apis/v1alpha1"
	"github.com/maximilianbraun/provider-matomo/internal/clients"
)

type caller interface {
	Call(ctx context.Context, method string, params url.Values) (json.RawMessage, error)
}

// Setup wires the Segment controller.
func Setup(mgr ctrl.Manager, cache *clients.Cache) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("segment").
		For(&v1alpha1.Segment{}).
		Complete(managed.NewReconciler(mgr,
			xpresource.ManagedKind(v1alpha1.SegmentGroupVersionKind),
			managed.WithTypedExternalConnector[*v1alpha1.Segment](&connector{kube: mgr.GetClient(), cache: cache}),
		))
}

type connector struct {
	kube  client.Client
	cache *clients.Cache
}

func (c *connector) Connect(ctx context.Context, mg *v1alpha1.Segment) (managed.TypedExternalClient[*v1alpha1.Segment], error) {
	pc := &v1alpha1.ProviderConfig{}
	if err := c.kube.Get(ctx, client.ObjectKey{Name: mg.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, fmt.Errorf("getting ProviderConfig: %w", err)
	}
	cl, err := c.cache.GetOrBuild(ctx, c.kube, pc)
	if err != nil {
		return nil, fmt.Errorf("building Matomo client: %w", err)
	}
	return &external{api: cl}, nil
}

type external struct {
	api caller
}

// matomoSegment mirrors the subset of SegmentEditor.get we reconcile.
type matomoSegment struct {
	Name             string      `json:"name"`
	Definition       string      `json:"definition"`
	EnableOnlyIDSite json.Number `json:"enable_only_idsite"`
	AutoArchive      json.Number `json:"auto_archive"`
	EnableAllUsers   json.Number `json:"enable_all_users"`
}

func (e *external) Observe(ctx context.Context, mg *v1alpha1.Segment) (managed.ExternalObservation, error) {
	id := meta.GetExternalName(mg)
	if id == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	params := url.Values{}
	params.Set("idSegment", id)
	raw, err := e.api.Call(ctx, "SegmentEditor.get", params)
	if err != nil {
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, fmt.Errorf("observing segment %s: %w", id, err)
	}

	var s matomoSegment
	if err := json.Unmarshal(raw, &s); err != nil {
		return managed.ExternalObservation{}, fmt.Errorf("decoding segment %s: %w", id, err)
	}

	if idInt, convErr := strconv.Atoi(id); convErr == nil {
		mg.Status.AtProvider.IDSegment = idInt
	}
	mg.Status.AtProvider.Definition = s.Definition
	mg.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: segmentUpToDate(mg.Spec.ForProvider, s),
	}, nil
}

func (e *external) Create(ctx context.Context, mg *v1alpha1.Segment) (managed.ExternalCreation, error) {
	raw, err := e.api.Call(ctx, "SegmentEditor.add", segmentParams(mg.Spec.ForProvider))
	if err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("creating segment: %w", err)
	}
	id, err := clients.DecodeID(raw)
	if err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("decoding new segment id: %w", err)
	}
	meta.SetExternalName(mg, strconv.Itoa(id))
	mg.Status.AtProvider.IDSegment = id
	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg *v1alpha1.Segment) (managed.ExternalUpdate, error) {
	id := meta.GetExternalName(mg)
	params := segmentParams(mg.Spec.ForProvider)
	params.Set("idSegment", id)
	if _, err := e.api.Call(ctx, "SegmentEditor.update", params); err != nil {
		return managed.ExternalUpdate{}, fmt.Errorf("updating segment %s: %w", id, err)
	}
	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg *v1alpha1.Segment) (managed.ExternalDelete, error) {
	id := meta.GetExternalName(mg)
	if id == "" {
		return managed.ExternalDelete{}, nil
	}
	params := url.Values{}
	params.Set("idSegment", id)
	if _, err := e.api.Call(ctx, "SegmentEditor.delete", params); err != nil && !clients.IsNotFound(err) {
		return managed.ExternalDelete{}, fmt.Errorf("deleting segment %s: %w", id, err)
	}
	return managed.ExternalDelete{}, nil
}

func (e *external) Disconnect(_ context.Context) error { return nil }

func segmentParams(p v1alpha1.SegmentParameters) url.Values {
	v := url.Values{}
	v.Set("name", p.Name)
	v.Set("definition", p.Definition)
	if p.IDSite != nil {
		v.Set("enable_only_idsite", strconv.Itoa(*p.IDSite))
	}
	if p.AutoArchive != nil {
		v.Set("auto_archive", b2s(*p.AutoArchive))
	}
	if p.EnabledAllUsers != nil {
		v.Set("enable_all_users", b2s(*p.EnabledAllUsers))
	}
	return v
}

func segmentUpToDate(p v1alpha1.SegmentParameters, s matomoSegment) bool {
	if p.Name != s.Name {
		return false
	}
	if p.Definition != s.Definition {
		return false
	}
	if p.IDSite != nil && strconv.Itoa(*p.IDSite) != s.EnableOnlyIDSite.String() {
		return false
	}
	if p.AutoArchive != nil && b2s(*p.AutoArchive) != s.AutoArchive.String() {
		return false
	}
	if p.EnabledAllUsers != nil && b2s(*p.EnabledAllUsers) != s.EnableAllUsers.String() {
		return false
	}
	return true
}

func b2s(b bool) string {
	if b {
		return "1"
	}
	return "0"
}
