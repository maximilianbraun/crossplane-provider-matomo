package controller

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/maximilianbraun/crossplane-provider-matomo/internal/clients"
	"github.com/maximilianbraun/crossplane-provider-matomo/internal/controller/providerconfig"
	"github.com/maximilianbraun/crossplane-provider-matomo/internal/controller/segment"
	"github.com/maximilianbraun/crossplane-provider-matomo/internal/controller/site"
	"github.com/maximilianbraun/crossplane-provider-matomo/internal/controller/token"
)

// Setup registers all controllers with the Manager.
func Setup(mgr ctrl.Manager, cache *clients.Cache) error {
	if err := providerconfig.Setup(mgr); err != nil {
		return err
	}

	for _, setup := range []func(ctrl.Manager, *clients.Cache) error{
		site.Setup,
		segment.Setup,
		token.Setup,
	} {
		if err := setup(mgr, cache); err != nil {
			return err
		}
	}
	return nil
}
