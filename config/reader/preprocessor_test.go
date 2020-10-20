package reader

import (
	"os"
	"strings"
	"testing"
)

func TestReplaceEnvVars(t *testing.T) {
	os.Setenv("myBar", "cat")
	os.Setenv("MYBAR", "cat")
	os.Setenv("my_Bar", "cat")
	os.Setenv("myBar_", "cat")

	testData := []struct {
		expected string
		data     []byte
	}{
		// Right use cases
		{
			`{"foo": "bar", "baz": {"bar": "cat"}}`,
			[]byte(`{"foo": "bar", "baz": {"bar": "${myBar}"}}`),
		},
		{
			`{"foo": "bar", "baz": {"bar": "cat"}}`,
			[]byte(`{"foo": "bar", "baz": {"bar": "${MYBAR}"}}`),
		},
		{
			`{"foo": "bar", "baz": {"bar": "cat"}}`,
			[]byte(`{"foo": "bar", "baz": {"bar": "${my_Bar}"}}`),
		},
		{
			`{"foo": "bar", "baz": {"bar": "cat"}}`,
			[]byte(`{"foo": "bar", "baz": {"bar": "${myBar_}"}}`),
		},
		// Wrong use cases
		{
			`{"foo": "bar", "baz": {"bar": "${myBar-}"}}`,
			[]byte(`{"foo": "bar", "baz": {"bar": "${myBar-}"}}`),
		},
		{
			`{"foo": "bar", "baz": {"bar": "${}"}}`,
			[]byte(`{"foo": "bar", "baz": {"bar": "${}"}}`),
		},
		{
			`{"foo": "bar", "baz": {"bar": "$sss}"}}`,
			[]byte(`{"foo": "bar", "baz": {"bar": "$sss}"}}`),
		},
		{
			`{"foo": "bar", "baz": {"bar": "${sss"}}`,
			[]byte(`{"foo": "bar", "baz": {"bar": "${sss"}}`),
		},
		{
			`{"foo": "bar", "baz": {"bar": "{something}"}}`,
			[]byte(`{"foo": "bar", "baz": {"bar": "{something}"}}`),
		},
		// Use cases without replace env vars
		{
			`{"foo": "bar", "baz": {"bar": "cat"}}`,
			[]byte(`{"foo": "bar", "baz": {"bar": "cat"}}`),
		},
	}

	for _, test := range testData {
		res, err := ReplaceEnvVars(test.data)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Compare(test.expected, string(res)) != 0 {
			t.Fatalf("Expected %s got %s", test.expected, res)
		}
	}
}
