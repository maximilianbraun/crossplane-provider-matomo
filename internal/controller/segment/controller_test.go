package segment

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/maximilianbraun/crossplane-provider-matomo/apis/v1alpha1"
)

type fakeCaller struct {
	calls []string
	last  url.Values
	resp  map[string]json.RawMessage
}

func (f *fakeCaller) Call(_ context.Context, method string, params url.Values) (json.RawMessage, error) {
	f.calls = append(f.calls, method)
	f.last = params
	return f.resp[method], nil
}

func newSeg(name, def string) *v1alpha1.Segment {
	return &v1alpha1.Segment{
		ObjectMeta: metav1.ObjectMeta{Name: "seg"},
		Spec: v1alpha1.SegmentSpec{ForProvider: v1alpha1.SegmentParameters{
			Name:       name,
			Definition: def,
		}},
	}
}

func TestObserve_UpToDate(t *testing.T) {
	mg := newSeg("DE returning", "visitCountry==de")
	meta.SetExternalName(mg, "5")
	fc := &fakeCaller{resp: map[string]json.RawMessage{
		"SegmentEditor.get": json.RawMessage(`{"name":"DE returning","definition":"visitCountry==de"}`),
	}}
	e := &external{api: fc}
	obs, err := e.Observe(context.Background(), mg)
	if err != nil {
		t.Fatal(err)
	}
	if !obs.ResourceExists || !obs.ResourceUpToDate {
		t.Fatalf("want exists+uptodate got %+v", obs)
	}
	if mg.Status.AtProvider.IDSegment != 5 {
		t.Fatalf("id not set: %d", mg.Status.AtProvider.IDSegment)
	}
}

func TestObserve_DefinitionDrift(t *testing.T) {
	mg := newSeg("DE returning", "visitCountry==de")
	meta.SetExternalName(mg, "5")
	fc := &fakeCaller{resp: map[string]json.RawMessage{
		"SegmentEditor.get": json.RawMessage(`{"name":"DE returning","definition":"visitCountry==fr"}`),
	}}
	e := &external{api: fc}
	obs, _ := e.Observe(context.Background(), mg)
	if obs.ResourceUpToDate {
		t.Fatal("expected drift on definition change")
	}
}

func TestCreate_SetsExternalName(t *testing.T) {
	mg := newSeg("DE", "visitCountry==de")
	fc := &fakeCaller{resp: map[string]json.RawMessage{
		"SegmentEditor.add": json.RawMessage(`11`),
	}}
	e := &external{api: fc}
	if _, err := e.Create(context.Background(), mg); err != nil {
		t.Fatal(err)
	}
	if meta.GetExternalName(mg) != "11" {
		t.Fatalf("external-name not set: %q", meta.GetExternalName(mg))
	}
	if fc.last.Get("definition") != "visitCountry==de" {
		t.Fatalf("params wrong: %v", fc.last)
	}
}

func TestDelete_CallsDelete(t *testing.T) {
	mg := newSeg("DE", "x")
	meta.SetExternalName(mg, "5")
	fc := &fakeCaller{}
	e := &external{api: fc}
	if _, err := e.Delete(context.Background(), mg); err != nil {
		t.Fatal(err)
	}
	if len(fc.calls) != 1 || fc.calls[0] != "SegmentEditor.delete" || fc.last.Get("idSegment") != "5" {
		t.Fatalf("delete not called right: %v %v", fc.calls, fc.last)
	}
}
