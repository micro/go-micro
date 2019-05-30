package json

import (
	"testing"

	"github.com/micro/go-micro/config/source"
)

func TestValues(t *testing.T) {
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

	values, err := newValues(&source.ChangeSet{
		Data: data,
	})

	if err != nil {
		t.Fatal(err)
	}

	for _, test := range testData {
		if v := values.Get(test.path...).String(""); v != test.value {
			t.Fatalf("Expected %s got %s for path %v", test.value, v, test.path)
		}
	}
}
