package token

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

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

// Setup wires the Token controller.
func Setup(mgr ctrl.Manager, cache *clients.Cache) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("token").
		For(&v1alpha1.Token{}).
		Complete(managed.NewReconciler(mgr,
			xpresource.ManagedKind(v1alpha1.TokenGroupVersionKind),
			managed.WithTypedExternalConnector[*v1alpha1.Token](&connector{kube: mgr.GetClient(), cache: cache}),
		))
}

type connector struct {
	kube  client.Client
	cache *clients.Cache
}

func (c *connector) Connect(ctx context.Context, mg *v1alpha1.Token) (managed.TypedExternalClient[*v1alpha1.Token], error) {
	pc := &v1alpha1.ProviderConfig{}
	if err := c.kube.Get(ctx, client.ObjectKey{Name: mg.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, fmt.Errorf("getting ProviderConfig: %w", err)
	}
	cl, err := c.cache.GetOrBuild(ctx, c.kube, pc)
	if err != nil {
		return nil, fmt.Errorf("building Matomo client: %w", err)
	}
	return &external{api: cl, kube: c.kube}, nil
}

type external struct {
	api  caller
	kube client.Client
}

// Observe cannot read a token back: Matomo stores tokens hashed and returns
// the value only once at creation. Existence is therefore inferred from the
// external name (set on Create), and an existing token is always reported
// up-to-date. Changing any field forces delete-and-recreate; since Delete is
// a no-op (see Delete), a field change will mint a new token without revoking
// the old one — document this for operators.
func (e *external) Observe(ctx context.Context, mg *v1alpha1.Token) (managed.ExternalObservation, error) {
	if meta.GetExternalName(mg) == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}
	mg.SetConditions(xpv1.Available())
	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, nil
}

func (e *external) Create(ctx context.Context, mg *v1alpha1.Token) (managed.ExternalCreation, error) {
	p := mg.Spec.ForProvider
	password, err := clients.ReadSecretValue(ctx, e.kube, p.PasswordConfirmationSecretRef)
	if err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("reading password confirmation: %w", err)
	}

	params := url.Values{}
	params.Set("userLogin", p.UserLogin)
	params.Set("passwordConfirmation", password)
	params.Set("description", p.Description)
	if p.ExpireHours != nil {
		params.Set("expireHours", strconv.FormatInt(*p.ExpireHours, 10))
	}

	raw, err := e.api.Call(ctx, "UsersManager.createAppSpecificTokenAuth", params)
	if err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("creating token: %w", err)
	}
	tokenValue, err := clients.DecodeString(raw)
	if err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("decoding token: %w", err)
	}

	meta.SetExternalName(mg, p.Description)
	mg.Status.AtProvider.CreatedAt = time.Now().UTC().Format(time.RFC3339)

	return managed.ExternalCreation{
		ConnectionDetails: managed.ConnectionDetails{
			"token":     []byte(tokenValue),
			"userLogin": []byte(p.UserLogin),
		},
	}, nil
}

// Update is unreachable in practice: Observe always reports up-to-date for an
// existing token.
func (e *external) Update(_ context.Context, _ *v1alpha1.Token) (managed.ExternalUpdate, error) {
	return managed.ExternalUpdate{}, nil
}

// Delete is intentionally a no-op. Matomo exposes no API to delete a single
// app-specific token by value or ID (only UsersManager.deleteAllTokensForUser,
// which would revoke every token the user holds). Deleting the Token managed
// resource therefore leaves the minted token valid until its expiry; operators
// who need immediate revocation must rotate the user's tokens out of band.
func (e *external) Delete(_ context.Context, _ *v1alpha1.Token) (managed.ExternalDelete, error) {
	return managed.ExternalDelete{}, nil
}

func (e *external) Disconnect(_ context.Context) error { return nil }
