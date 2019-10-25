package metadata

import (
	"context"
	"testing"
)

func TestMetadataCopy(t *testing.T) {
	md := Metadata{
		"foo": "bar",
		"bar": "baz",
	}

	cp := Copy(md)

	for k, v := range md {
		if cv := cp[k]; cv != v {
			t.Fatalf("Got %s:%s for %s:%s", k, cv, k, v)
		}
	}
}

func TestMetadataContext(t *testing.T) {
	md := Metadata{
		"foo": "bar",
	}

	ctx := NewContext(context.TODO(), md)

	emd, ok := FromContext(ctx)
	if !ok {
		t.Errorf("Unexpected error retrieving metadata, got %t", ok)
	}

	if emd["foo"] != md["foo"] {
		t.Errorf("Expected key: %s val: %s, got key: %s val: %s", "foo", md["foo"], "foo", emd["foo"])
	}

	if i := len(emd); i != 1 {
		t.Errorf("Expected metadata length 1 got %d", i)
	}
}
func TestPatchContext(t *testing.T) {

	original := Metadata{
		"foo": "bar",
	}

	patch := Metadata{
		"sumo": "demo",
	}
	ctx := NewContext(context.TODO(), original)

	patchedCtx := PatchContext(ctx, patch)

	patchedMd, ok := FromContext(patchedCtx)
	if !ok {
		t.Errorf("Unexpected error retrieving metadata, got %t", ok)
	}

	if patchedMd["sumo"] != patch["sumo"] {
		t.Errorf("Expected key: %s val: %s, got key: %s val: %s", "sumo", patch["sumo"], "sumo", patchedMd["sumo"])
	}
	if patchedMd["foo"] != original["foo"] {
		t.Errorf("Expected key: %s val: %s, got key: %s val: %s", "foo", original["foo"], "foo", patchedMd["foo"])
	}
}
