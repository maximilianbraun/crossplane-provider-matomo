//go:build e2e

package site

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/maximilianbraun/provider-matomo/apis/v1alpha1"
	"github.com/maximilianbraun/provider-matomo/internal/clients"
)

func e2eExternal(t *testing.T) *external {
	t.Helper()
	url, token := os.Getenv("MATOMO_URL"), os.Getenv("MATOMO_TOKEN")
	if url == "" || token == "" {
		t.Skip("MATOMO_URL/MATOMO_TOKEN not set; skipping e2e")
	}
	return &external{api: clients.NewClient(url, token)}
}

func TestE2E_SiteLifecycle(t *testing.T) {
	ctx := context.Background()
	e := e2eExternal(t)
	tag := fmt.Sprintf("e2e-%d", time.Now().UnixNano())

	mg := &v1alpha1.Site{
		ObjectMeta: metav1.ObjectMeta{Name: "e2e-site"},
		Spec: v1alpha1.SiteSpec{ForProvider: v1alpha1.SiteParameters{
			SiteName: tag,
			URLs:     []string{"https://e2e.example.com"},
			Type:     "website",
		}},
	}

	if _, err := e.Create(ctx, mg); err != nil {
		t.Fatalf("create: %v", err)
	}
	id := meta.GetExternalName(mg)
	if id == "" {
		t.Fatal("create did not set external-name")
	}
	t.Cleanup(func() { _, _ = e.Delete(ctx, mg) })

	obs, err := e.Observe(ctx, mg)
	if err != nil || !obs.ResourceExists || !obs.ResourceUpToDate {
		t.Fatalf("post-create observe: %+v err=%v", obs, err)
	}

	// Introduce drift by renaming, expect Observe to report not-up-to-date.
	mg.Spec.ForProvider.SiteName = tag + "-renamed"
	obs, err = e.Observe(ctx, mg)
	if err != nil || !obs.ResourceExists || obs.ResourceUpToDate {
		t.Fatalf("drift observe: %+v err=%v", obs, err)
	}

	if _, err := e.Update(ctx, mg); err != nil {
		t.Fatalf("update: %v", err)
	}
	obs, err = e.Observe(ctx, mg)
	if err != nil || !obs.ResourceUpToDate {
		t.Fatalf("post-update observe: %+v err=%v", obs, err)
	}

	if _, err := e.Delete(ctx, mg); err != nil {
		t.Fatalf("delete: %v", err)
	}
	obs, err = e.Observe(ctx, mg)
	if err != nil || obs.ResourceExists {
		t.Fatalf("post-delete observe should be gone: %+v err=%v", obs, err)
	}
}
