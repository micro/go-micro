package vault

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/asim/go-micro/v3/config"
)

func TestVaultMakeMap(t *testing.T) {
	tt := []struct {
		name       string
		expected   []byte
		input      []byte
		secretName string
	}{
		{
			name:       "simple valid data 1",
			secretName: "my/secret",
			input:      []byte(`{"data":{"bar":"bazz", "tar":"par"}, "metadata":{"version":1, "destroyed": false}}`),
			expected:   []byte(`{"my":{"secret":{"bar":"bazz", "tar":"par"}}}`),
		},
		{
			name:       "simple valid data 2",
			secretName: "my/secret",
			input:      []byte(`{"bar":"bazz", "tar":"par"}`),
			expected:   []byte(`{"my":{"secret":{"bar":"bazz", "tar":"par"}}}`),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			var input map[string]interface{}
			var expected map[string]interface{}

			_ = json.Unmarshal(tc.input, &input)
			_ = json.Unmarshal(tc.expected, &expected)

			out, _ := makeMap(input, tc.secretName)

			if eq := reflect.DeepEqual(out, expected); !eq {
				fmt.Println(eq)
				t.Fatalf("expected %v and got %v", expected, out)
			}
		})
	}
}

func TestVault_Read(t *testing.T) {
	if tr := os.Getenv("TRAVIS"); len(tr) > 0 {
		t.Skip()
	}

	var (
		address  = "http://127.0.0.1"
		resource = "secret/data/db/auth"
		token    = "s.Q4Zi0CSowXZl7sh0z96ijcT4"
	)

	data := []byte(`{"secret":{"data":{"db":{"auth":{"host":"128.23.33.21","password":"mypassword","port":"3306","user":"myuser"}}}}}`)

	tt := []struct {
		name     string
		addr     string
		resource string
		token    string
	}{
		{name: "read data basic", addr: address, resource: resource, token: token},
		{name: "read data without token", addr: address, resource: resource, token: ""},
		{name: "read data full address format", addr: "http://127.0.0.1:8200", resource: resource, token: token},
		{name: "read data wrong resource path", addr: address, resource: "secrets/data/db/auth", token: token},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			source := NewSource(
				WithAddress(tc.addr),
				WithResourcePath(tc.resource),
				WithToken(tc.token),
			)

			r, err := source.Read()
			if err != nil {
				if tc.token == "" {
					return
				} else if strings.Compare(err.Error(), "source not found: secrets/data/db/auth") == 0 {
					return
				}
				t.Errorf("%s: not able to read the config values because: %v", tc.name, err)
				return
			}

			if string(r.Data) != string(data) {
				t.Logf("data expected: %v", string(data))
				t.Logf("data got from configmap: %v", string(r.Data))
				t.Errorf("data from configmap does not match.")
			}
		})
	}
}

func TestVault_String(t *testing.T) {
	source := NewSource()

	if source.String() != "vault" {
		t.Errorf("expecting to get %v and instead got %v", "vault", source)
	}
}

func TestVaultNewSource(t *testing.T) {
	if tr := os.Getenv("TRAVIS"); len(tr) > 0 {
		t.Skip()
	}

	conf, err := config.NewConfig()
	if err != nil {
		t.Fatal(err)
	}

	_ = conf.Load(NewSource(
		WithAddress("http://127.0.0.1"),
		WithResourcePath("secret/data/db/auth"),
		WithToken("s.Q4Zi0CSowXZl7sh0z96ijcT4"),
	))

	if user := conf.Get("secret", "data", "db", "auth", "user").String("user"); user != "myuser" {
		t.Errorf("expected %v and got %v", "myuser", user)
	}

	if addr := conf.Get("secret", "data", "db", "auth", "host").String("host"); addr != "128.23.33.21" {
		t.Errorf("expected %v and got %v", "128.23.33.21", addr)
	}
}
