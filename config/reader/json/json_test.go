package json

import (
	"testing"

	"github.com/micro/go-micro/v3/config/reader"
	"github.com/micro/go-micro/v3/config/source"
)

func TestReader(t *testing.T) {
	data := []byte(`{"foo": "bar", "baz": {"bar": "cat"}}`)

	testData := []struct {
		path  []string
		value string
	}{
		{
			[]string{"foo"},
			"bar",
		},
		{
			[]string{"baz", "bar"},
			"cat",
		},
	}

	values := newTestValues(t, data)

	for _, test := range testData {
		if v := values.Get(test.path...).String(""); v != test.value {
			t.Fatalf("Expected %s got %s for path %v", test.value, v, test.path)
		}
	}
}

func TestDisableReplaceEnvVars(t *testing.T) {
	data := []byte(`{"foo": "bar", "baz": {"bar": "test/${test}"}}`)

	tests := []struct {
		path  []string
		value string
		opts  []reader.Option
	}{
		{
			[]string{"baz", "bar"},
			"test/",
			nil,
		},
		{
			[]string{"baz", "bar"},
			"test/${test}",
			[]reader.Option{reader.WithDisableReplaceEnvVars()},
		},
	}

	for _, test := range tests {
		values := newTestValues(t, data, test.opts...)

		if v := values.Get(test.path...).String(""); v != test.value {
			t.Fatalf("Expected %s got %s for path %v", test.value, v, test.path)
		}
	}
}

func newTestValues(t *testing.T, data []byte, opts ...reader.Option) reader.Values {
	r := NewReader(opts...)

	c, err := r.Merge(&source.ChangeSet{Data: data}, &source.ChangeSet{})
	if err != nil {
		t.Fatal(err)
	}

	values, err := r.Values(c)
	if err != nil {
		t.Fatal(err)
	}

	return values
}
