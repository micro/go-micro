package configmap

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/asim/go-micro/v3/config"
)

func TestGetClient(t *testing.T) {
	if tr := os.Getenv("TRAVIS"); len(tr) > 0 {
		return
	}

	localCfg := os.Getenv("HOME") + "/.kube/config"
	tt := []struct {
		name        string
		cfgpath     string
		error       string
		isIncluster bool
		assert      string
	}{
		{name: "fail loading incluster kubeconfig, from external", error: "unable to load in-cluster configuration, KUBERNETES_SERVICE_HOST and KUBERNETES_SERVICE_PORT must be defined", isIncluster: false},
		{name: "fail loading external kubeconfig", cfgpath: "/invalid/path", error: "stat /invalid/path: no such file or directory", isIncluster: false},
		{name: "loading an incluster kubeconfig", cfgpath: "", error: "", isIncluster: true, assert: "open /var/run/secrets/kubernetes.io/serviceaccount/token: no such file or directory"},
		{name: "loading kubeconfig from external", cfgpath: localCfg, isIncluster: false},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {

			if tc.isIncluster {
				os.Setenv("KUBERNETES_SERVICE_HOST", "127.0.0.1")
				os.Setenv("KUBERNETES_SERVICE_PORT", "443")
			} else {
				os.Unsetenv("KUBERNETES_SERVICE_HOST")
				os.Unsetenv("KUBERNETES_SERVICE_PORT")
			}

			_, err := getClient(tc.cfgpath)

			if err != nil {
				if err.Error() == tc.error {
					return
				}
				if err.Error() == tc.assert {
					return
				}

				t.Errorf("found an unhandled error: %v", err)
			}
		})
	}

	os.Unsetenv("KUBERNETES_SERVICE_HOST")
	os.Unsetenv("KUBERNETES_SERVICE_PORT")
}

func TestMakeMap(t *testing.T) {
	if tr := os.Getenv("TRAVIS"); len(tr) > 0 {
		return
	}

	tt := []struct {
		name  string
		din   map[string]string
		dout  map[string]interface{}
		jdout []byte
	}{
		{
			name: "simple valid data",
			din: map[string]string{
				"foo": "bar=bazz",
			},
			dout: map[string]interface{}{
				"foo": map[string]interface{}{
					"bar": "bazz",
				},
			},
			jdout: []byte(`{"foo":{"bar":"bazz"}}`),
		},
		{
			name: "complex valid data",
			din: map[string]string{
				"mongodb": "host=127.0.0.1\nport=27017\nuser=user\npassword=password",
				"config":  "host=0.0.0.0\nport=1337",
				"redis":   "url=redis://127.0.0.1:6379/db01",
				"sql":     "username=user\npassword=password=1234",
			},
			dout: map[string]interface{}{
				"mongodb": map[string]interface{}{
					"host":     "127.0.0.1",
					"port":     "27017",
					"user":     "user",
					"password": "password",
				},
				"config": map[string]interface{}{
					"host": "0.0.0.0",
					"port": "1337",
				},
				"redis": map[string]interface{}{
					"url": "redis://127.0.0.1:6379/db01",
				},
				"sql": map[string]interface{}{
					"username": "user",
					"password": "password=1234",
				},
			},
			jdout: []byte(`{"config":{"host":"0.0.0.0","port":"1337"},"mongodb":{"host":"127.0.0.1","password":"password","port":"27017","user":"user"},"redis":{"url":"redis://127.0.0.1:6379/db01"},"sql":{"password":"password=1234","username":"user"}}`),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			dout := makeMap(tc.din)
			jdout, _ := json.Marshal(dout)

			if eq := reflect.DeepEqual(dout, tc.dout); !eq {
				fmt.Println(eq)
				t.Fatalf("expected %v and got %v", tc.dout, dout)
			}

			if string(jdout) != string(tc.jdout) {
				t.Fatalf("expected %v and got %v", string(tc.jdout), string(jdout))
			}
		})
	}
}

func TestConfigmap_Read(t *testing.T) {
	if tr := os.Getenv("TRAVIS"); len(tr) > 0 {
		return
	}

	data := []byte(`{"config":{"host":"0.0.0.0","port":"1337"},"mongodb":{"host":"127.0.0.1","password":"password","port":"27017","user":"user"},"redis":{"url":"redis://127.0.0.1:6379/db01"}}`)

	tt := []struct {
		name      string
		sname     string
		namespace string
	}{
		{name: "read data with source default values", sname: DefaultName, namespace: DefaultNamespace},
		{name: "read data with source with custom configmap name", sname: "micro-config", namespace: DefaultNamespace},
		{name: "read data with source with custom namespace", sname: DefaultName, namespace: "kube-public"},
		{name: "read data with source with custom configmap name and namespace", sname: "micro-config", namespace: "kube-public"},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			source := NewSource(
				WithName(tc.sname),
				WithNamespace(tc.namespace),
			)

			r, err := source.Read()
			if err != nil {
				t.Errorf("not able to read the config values because: %v", err)
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

func TestConfigmap_String(t *testing.T) {
	if tr := os.Getenv("TRAVIS"); len(tr) > 0 {
		return
	}

	source := NewSource()

	if source.String() != "configmap" {
		t.Errorf("expecting to get %v and instead got %v", "configmap", source)
	}
}

func TestNewSource(t *testing.T) {
	if tr := os.Getenv("TRAVIS"); len(tr) > 0 {
		return
	}

	conf, err := config.NewConfig()
	if err != nil {
		t.Fatal(err)
	}

	conf.Load(NewSource())

	if mongodbHost := conf.Get("mongodb", "host").String("localhost"); mongodbHost != "127.0.0.1" && mongodbHost != "localhost" {
		t.Errorf("expected %v and got %v", "127.0.0.1", mongodbHost)
	}

	if configPort := conf.Get("config", "port").Int(1337); configPort != 1337 {
		t.Errorf("expected %v and got %v", "1337", configPort)
	}
}
