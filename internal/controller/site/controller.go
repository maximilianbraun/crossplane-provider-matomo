package site

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	xpresource "github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/maximilianbraun/crossplane-provider-matomo/apis/v1alpha1"
	"github.com/maximilianbraun/crossplane-provider-matomo/internal/clients"
)

// caller is the subset of the Matomo client used by this controller. An
// interface keeps the external client unit-testable with a fake.
type caller interface {
	Call(ctx context.Context, method string, params url.Values) (json.RawMessage, error)
}

// Setup wires the Site controller.
func Setup(mgr ctrl.Manager, cache *clients.Cache) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("site").
		For(&v1alpha1.Site{}).
		Complete(managed.NewReconciler(mgr,
			xpresource.ManagedKind(v1alpha1.SiteGroupVersionKind),
			managed.WithTypedExternalConnector[*v1alpha1.Site](&connector{kube: mgr.GetClient(), cache: cache}),
		))
}

type connector struct {
	kube  client.Client
	cache *clients.Cache
}

func (c *connector) Connect(ctx context.Context, mg *v1alpha1.Site) (managed.TypedExternalClient[*v1alpha1.Site], error) {
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

// matomoSite mirrors the subset of SitesManager.getSiteFromId we reconcile.
// Matomo returns ecommerce/sitesearch as JSON numbers and the rest as strings.
type matomoSite struct {
	Name               string      `json:"name"`
	MainURL            string      `json:"main_url"`
	TsCreated          string      `json:"ts_created"`
	Ecommerce          json.Number `json:"ecommerce"`
	SiteSearch         json.Number `json:"sitesearch"`
	Timezone           string      `json:"timezone"`
	Currency           string      `json:"currency"`
	ExcludedIPs        string      `json:"excluded_ips"`
	ExcludedParameters string      `json:"excluded_parameters"`
	Type               string      `json:"type"`
}

func (e *external) Observe(ctx context.Context, mg *v1alpha1.Site) (managed.ExternalObservation, error) {
	id := meta.GetExternalName(mg)
	if id == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	params := url.Values{}
	params.Set("idSite", id)
	raw, err := e.api.Call(ctx, "SitesManager.getSiteFromId", params)
	if err != nil {
		if clients.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, fmt.Errorf("observing site %s: %w", id, err)
	}

	var s matomoSite
	if err := json.Unmarshal(raw, &s); err != nil {
		return managed.ExternalObservation{}, fmt.Errorf("decoding site %s: %w", id, err)
	}

	if idInt, convErr := strconv.Atoi(id); convErr == nil {
		mg.Status.AtProvider.IDSite = idInt
	}
	mg.Status.AtProvider.MainURL = s.MainURL
	mg.Status.AtProvider.TsCreated = s.TsCreated
	mg.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: siteUpToDate(mg.Spec.ForProvider, s),
	}, nil
}

func (e *external) Create(ctx context.Context, mg *v1alpha1.Site) (managed.ExternalCreation, error) {
	raw, err := e.api.Call(ctx, "SitesManager.addSite", siteParams(mg.Spec.ForProvider))
	if err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("creating site: %w", err)
	}
	id, err := clients.DecodeID(raw)
	if err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("decoding new site id: %w", err)
	}
	meta.SetExternalName(mg, strconv.Itoa(id))
	mg.Status.AtProvider.IDSite = id
	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg *v1alpha1.Site) (managed.ExternalUpdate, error) {
	id := meta.GetExternalName(mg)
	params := siteParams(mg.Spec.ForProvider)
	params.Set("idSite", id)
	if _, err := e.api.Call(ctx, "SitesManager.updateSite", params); err != nil {
		return managed.ExternalUpdate{}, fmt.Errorf("updating site %s: %w", id, err)
	}
	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg *v1alpha1.Site) (managed.ExternalDelete, error) {
	id := meta.GetExternalName(mg)
	if id == "" {
		return managed.ExternalDelete{}, nil
	}
	params := url.Values{}
	params.Set("idSite", id)
	if _, err := e.api.Call(ctx, "SitesManager.deleteSite", params); err != nil && !clients.IsNotFound(err) {
		return managed.ExternalDelete{}, fmt.Errorf("deleting site %s: %w", id, err)
	}
	return managed.ExternalDelete{}, nil
}

func (e *external) Disconnect(_ context.Context) error { return nil }

// siteParams builds the SitesManager add/update parameter set from spec.
func siteParams(p v1alpha1.SiteParameters) url.Values {
	v := url.Values{}
	v.Set("siteName", p.SiteName)
	for i, u := range p.URLs {
		v.Set(fmt.Sprintf("urls[%d]", i), u)
	}
	if p.Ecommerce != nil {
		v.Set("ecommerce", b2s(*p.Ecommerce))
	}
	if p.SiteSearch != nil {
		v.Set("siteSearch", b2s(*p.SiteSearch))
	}
	if p.Timezone != "" {
		v.Set("timezone", p.Timezone)
	}
	if p.Currency != "" {
		v.Set("currency", p.Currency)
	}
	if len(p.ExcludedIPs) > 0 {
		v.Set("excludedIps", strings.Join(p.ExcludedIPs, ","))
	}
	if len(p.ExcludedQueryParameters) > 0 {
		v.Set("excludedQueryParameters", strings.Join(p.ExcludedQueryParameters, ","))
	}
	if p.Type != "" {
		v.Set("type", p.Type)
	}
	return v
}

// siteUpToDate compares desired spec against observed Matomo state for the
// fields the provider manages. Only fields set in spec are compared.
func siteUpToDate(p v1alpha1.SiteParameters, s matomoSite) bool {
	if p.SiteName != s.Name {
		return false
	}
	if len(p.URLs) > 0 && p.URLs[0] != s.MainURL {
		return false
	}
	if p.Ecommerce != nil && b2s(*p.Ecommerce) != s.Ecommerce.String() {
		return false
	}
	if p.SiteSearch != nil && b2s(*p.SiteSearch) != s.SiteSearch.String() {
		return false
	}
	if p.Timezone != "" && p.Timezone != s.Timezone {
		return false
	}
	if p.Currency != "" && p.Currency != s.Currency {
		return false
	}
	if p.Type != "" && p.Type != s.Type {
		return false
	}
	if len(p.ExcludedIPs) > 0 && strings.Join(p.ExcludedIPs, ",") != s.ExcludedIPs {
		return false
	}
	if len(p.ExcludedQueryParameters) > 0 && strings.Join(p.ExcludedQueryParameters, ",") != s.ExcludedParameters {
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
