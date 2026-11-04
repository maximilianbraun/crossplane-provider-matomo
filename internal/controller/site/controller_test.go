package site

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/maximilianbraun/provider-matomo/apis/v1alpha1"
)

const (
	urlA       = "https://a"
	urlShop    = "https://shop.example.com"
	mGetSiteID = "SitesManager.getSiteFromId"
)

type fakeCaller struct {
	calls []string
	last  url.Values
	resp  map[string]json.RawMessage
	errs  map[string]error
}

func (f *fakeCaller) Call(_ context.Context, method string, params url.Values) (json.RawMessage, error) {
	f.calls = append(f.calls, method)
	f.last = params
	if e, ok := f.errs[method]; ok {
		return nil, e
	}
	return f.resp[method], nil
}

func newSite(name string, urls []string) *v1alpha1.Site {
	return &v1alpha1.Site{
		ObjectMeta: metav1.ObjectMeta{Name: "s"},
		Spec: v1alpha1.SiteSpec{ForProvider: v1alpha1.SiteParameters{
			SiteName: name,
			URLs:     urls,
		}},
	}
}

func TestObserve_NoExternalName(t *testing.T) {
	e := &external{api: &fakeCaller{}}
	obs, err := e.Observe(context.Background(), newSite("x", []string{urlA}))
	if err != nil || obs.ResourceExists {
		t.Fatalf("want exists=false,nil got %+v,%v", obs, err)
	}
}

func TestObserve_FoundUpToDate(t *testing.T) {
	mg := newSite("Shop", []string{urlShop})
	meta.SetExternalName(mg, "3")
	fc := &fakeCaller{resp: map[string]json.RawMessage{
		mGetSiteID: json.RawMessage(`{"name":"Shop","main_url":"https://shop.example.com","ts_created":"2020-01-01","type":"website"}`),
	}}
	e := &external{api: fc}
	obs, err := e.Observe(context.Background(), mg)
	if err != nil {
		t.Fatal(err)
	}
	if !obs.ResourceExists || !obs.ResourceUpToDate {
		t.Fatalf("want exists+uptodate, got %+v", obs)
	}
	if mg.Status.AtProvider.IDSite != 3 || mg.Status.AtProvider.MainURL != urlShop {
		t.Fatalf("status not populated: %+v", mg.Status.AtProvider)
	}
}

func TestObserve_Drift(t *testing.T) {
	mg := newSite("Shop", []string{urlShop})
	meta.SetExternalName(mg, "3")
	fc := &fakeCaller{resp: map[string]json.RawMessage{
		mGetSiteID: json.RawMessage(`{"name":"Old Name","main_url":"https://shop.example.com"}`),
	}}
	e := &external{api: fc}
	obs, _ := e.Observe(context.Background(), mg)
	if !obs.ResourceExists || obs.ResourceUpToDate {
		t.Fatalf("want exists + drift, got %+v", obs)
	}
}

func TestObserve_NotFound(t *testing.T) {
	mg := newSite("Shop", []string{urlA})
	meta.SetExternalName(mg, "9")
	fc := &fakeCaller{errs: map[string]error{
		mGetSiteID: errors.New("matomo API error: website id = '9' couldn't be found"),
	}}
	e := &external{api: fc}
	obs, err := e.Observe(context.Background(), mg)
	if err != nil || obs.ResourceExists {
		t.Fatalf("want exists=false,nil got %+v,%v", obs, err)
	}
}

func TestCreate_SetsExternalName(t *testing.T) {
	mg := newSite("Shop", []string{urlA, "https://b"})
	fc := &fakeCaller{resp: map[string]json.RawMessage{
		"SitesManager.addSite": json.RawMessage(`{"value":42}`),
	}}
	e := &external{api: fc}
	if _, err := e.Create(context.Background(), mg); err != nil {
		t.Fatal(err)
	}
	if meta.GetExternalName(mg) != "42" || mg.Status.AtProvider.IDSite != 42 {
		t.Fatalf("external-name/id not set: %q %d", meta.GetExternalName(mg), mg.Status.AtProvider.IDSite)
	}
	if fc.last.Get("siteName") != "Shop" || fc.last.Get("urls[1]") != "https://b" {
		t.Fatalf("params wrong: %v", fc.last)
	}
}

func TestDelete_CallsDeleteSite(t *testing.T) {
	mg := newSite("Shop", []string{urlA})
	meta.SetExternalName(mg, "3")
	fc := &fakeCaller{}
	e := &external{api: fc}
	if _, err := e.Delete(context.Background(), mg); err != nil {
		t.Fatal(err)
	}
	if len(fc.calls) != 1 || fc.calls[0] != "SitesManager.deleteSite" || fc.last.Get("idSite") != "3" {
		t.Fatalf("deleteSite not called correctly: %v %v", fc.calls, fc.last)
	}
}
