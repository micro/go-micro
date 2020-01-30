package json

import (
	"reflect"
	"testing"

	"github.com/micro/go-micro/v2/config/source"
)

func TestValues(t *testing.T) {
	emptyStr := ""
	testData := []struct {
		csdata   []byte
		path     []string
		accepter interface{}
		value    interface{}
	}{
		{
			[]byte(`{"foo": "bar", "baz": {"bar": "cat"}}`),
			[]string{"foo"},
			emptyStr,
			"bar",
		},
		{
			[]byte(`{"foo": "bar", "baz": {"bar": "cat"}}`),
			[]string{"baz", "bar"},
			emptyStr,
			"cat",
		},
	}

	for idx, test := range testData {
		values, err := newValues(&source.ChangeSet{
			Data: test.csdata,
		})
		if err != nil {
			t.Fatal(err)
		}

		err = values.Get(test.path...).Scan(&test.accepter)
		if err != nil {
			t.Fatal(err)
		}
		if test.accepter != test.value {
			t.Fatalf("No.%d Expected %v got %v for path %v", idx, test.value, test.accepter, test.path)
		}
	}
}

func TestStructArray(t *testing.T) {
	type T struct {
		Foo string
	}

	emptyTSlice := []T{}

	testData := []struct {
		csdata   []byte
		accepter []T
		value    []T
	}{
		{
			[]byte(`[{"foo": "bar"}]`),
			emptyTSlice,
			[]T{{Foo: "bar"}},
		},
	}

	for idx, test := range testData {
		values, err := newValues(&source.ChangeSet{
			Data: test.csdata,
		})
		if err != nil {
			t.Fatal(err)
		}

		err = values.Get().Scan(&test.accepter)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(test.accepter, test.value) {
			t.Fatalf("No.%d Expected %v got %v", idx, test.value, test.accepter)
		}
	}
}
