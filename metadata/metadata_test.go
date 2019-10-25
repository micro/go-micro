package metadata

import (
	"context"
	"reflect"
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

func TestMergeContext(t *testing.T) {
	type args struct {
		existing  Metadata
		append    Metadata
		overwrite bool
	}
	tests := []struct {
		name string
		args args
		want Metadata
	}{
		{
			name: "matching key, overwrite false",
			args: args{
				existing:  Metadata{"foo": "bar", "sumo": "demo"},
				append:    Metadata{"sumo": "demo2"},
				overwrite: false,
			},
			want: Metadata{"foo": "bar", "sumo": "demo"},
		},
		{
			name: "matching key, overwrite true",
			args: args{
				existing:  Metadata{"foo": "bar", "sumo": "demo"},
				append:    Metadata{"sumo": "demo2"},
				overwrite: true,
			},
			want: Metadata{"foo": "bar", "sumo": "demo2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, _ := FromContext(MergeContext(NewContext(context.TODO(), tt.args.existing), tt.args.append, tt.args.overwrite)); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MergeContext() = %v, want %v", got, tt.want)
			}
		})
	}
}
