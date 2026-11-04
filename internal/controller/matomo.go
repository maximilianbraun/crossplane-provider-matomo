package controller

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/maximilianbraun/provider-matomo/internal/clients"
	"github.com/maximilianbraun/provider-matomo/internal/controller/providerconfig"
)

// Setup registers all controllers with the Manager.
func Setup(mgr ctrl.Manager, cache *clients.Cache) error {
	if err := providerconfig.Setup(mgr); err != nil {
		return err
	}

	// Resource controllers (Site, Segment, Token) are registered here once
	// implemented.
	_ = cache
	return nil
}
