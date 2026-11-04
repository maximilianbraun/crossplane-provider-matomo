package token

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"

	"github.com/maximilianbraun/crossplane-provider-matomo/apis/v1alpha1"
)

type fakeCaller struct {
	calls []string
	last  url.Values
	resp  json.RawMessage
}

func (f *fakeCaller) Call(_ context.Context, method string, params url.Values) (json.RawMessage, error) {
	f.calls = append(f.calls, method)
	f.last = params
	return f.resp, nil
}

func newToken() *v1alpha1.Token {
	return &v1alpha1.Token{
		ObjectMeta: metav1.ObjectMeta{Name: "tok"},
		Spec: v1alpha1.TokenSpec{ForProvider: v1alpha1.TokenParameters{
			UserLogin:   "ci",
			Description: "CI token",
			PasswordConfirmationSecretRef: xpv1.SecretKeySelector{
				SecretReference: xpv1.SecretReference{Name: "pw", Namespace: "crossplane-system"},
				Key:             "password",
			},
		}},
	}
}

func TestObserve_Existence(t *testing.T) {
	e := &external{api: &fakeCaller{}}

	obs, _ := e.Observe(context.Background(), newToken())
	if obs.ResourceExists {
		t.Fatal("no external-name must mean not exist")
	}

	mg := newToken()
	meta.SetExternalName(mg, "CI token")
	obs, _ = e.Observe(context.Background(), mg)
	if !obs.ResourceExists || !obs.ResourceUpToDate {
		t.Fatalf("existing token must be exists+uptodate: %+v", obs)
	}
}

func TestCreate_MintsAndPublishes(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "pw", Namespace: "crossplane-system"},
		Data:       map[string][]byte{"password": []byte("s3cret")},
	}
	kube := fake.NewClientBuilder().WithObjects(secret).Build()
	fc := &fakeCaller{resp: json.RawMessage(`"abcdef123456"`)}
	e := &external{api: fc, kube: kube}

	mg := newToken()
	out, err := e.Create(context.Background(), mg)
	if err != nil {
		t.Fatal(err)
	}
	if meta.GetExternalName(mg) != "CI token" {
		t.Fatalf("external-name not set: %q", meta.GetExternalName(mg))
	}
	if string(out.ConnectionDetails["token"]) != "abcdef123456" {
		t.Fatalf("token not published: %q", out.ConnectionDetails["token"])
	}
	if fc.last.Get("passwordConfirmation") != "s3cret" || fc.last.Get("userLogin") != "ci" {
		t.Fatalf("params wrong: %v", fc.last)
	}
	if mg.Status.AtProvider.CreatedAt == "" {
		t.Fatal("CreatedAt not set")
	}
}

func TestDelete_NoOp(t *testing.T) {
	fc := &fakeCaller{}
	e := &external{api: fc}
	mg := newToken()
	meta.SetExternalName(mg, "CI token")
	if _, err := e.Delete(context.Background(), mg); err != nil {
		t.Fatal(err)
	}
	if len(fc.calls) != 0 {
		t.Fatalf("delete must not call Matomo, got %v", fc.calls)
	}
}
