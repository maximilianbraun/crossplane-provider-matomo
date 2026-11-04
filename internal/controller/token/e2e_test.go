//go:build e2e

package token

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/maximilianbraun/crossplane-provider-matomo/apis/v1alpha1"
	"github.com/maximilianbraun/crossplane-provider-matomo/internal/clients"
)

func TestE2E_TokenMintAuthenticates(t *testing.T) {
	ctx := context.Background()
	matomoURL, superToken := os.Getenv("MATOMO_URL"), os.Getenv("MATOMO_TOKEN")
	adminPass := os.Getenv("MATOMO_ADMIN_PASS")
	if matomoURL == "" || superToken == "" || adminPass == "" {
		t.Skip("MATOMO_URL/MATOMO_TOKEN/MATOMO_ADMIN_PASS not set; skipping e2e")
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "admin-pw", Namespace: "default"},
		Data:       map[string][]byte{"password": []byte(adminPass)},
	}
	kube := fake.NewClientBuilder().WithObjects(secret).Build()

	e := &external{api: clients.NewClient(matomoURL, superToken), kube: kube}

	mg := &v1alpha1.Token{
		ObjectMeta: metav1.ObjectMeta{Name: "e2e-token"},
		Spec: v1alpha1.TokenSpec{ForProvider: v1alpha1.TokenParameters{
			UserLogin:   "admin",
			Description: fmt.Sprintf("e2e-%d", time.Now().UnixNano()),
			PasswordConfirmationSecretRef: xpv1.SecretKeySelector{
				SecretReference: xpv1.SecretReference{Name: "admin-pw", Namespace: "default"},
				Key:             "password",
			},
		}},
	}

	out, err := e.Create(ctx, mg)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	minted := string(out.ConnectionDetails["token"])
	if minted == "" {
		t.Fatal("no token in connection details")
	}

	// The minted token must authenticate a real Reporting API call.
	probe := clients.NewClient(matomoURL, minted)
	raw, err := probe.Call(ctx, "API.getMatomoVersion", url.Values{})
	if err != nil {
		t.Fatalf("minted token failed to authenticate: %v", err)
	}
	if v, err := clients.DecodeString(raw); err != nil || v == "" {
		t.Fatalf("unexpected version response %q err=%v", string(raw), err)
	}
}
