package configmap

import (
	"strings"

	// "strings"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func getClient(configPath string) (*kubernetes.Clientset, error) {
	var config *rest.Config
	var err error

	if configPath == "" {
		config, err = rest.InClusterConfig()
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", configPath)
	}

	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}

func makeMap(kv map[string]string) map[string]interface{} {

	data := make(map[string]interface{})

	for k, v := range kv {
		data[k] = make(map[string]interface{})

		vals := strings.Split(v, "\n")

		mp := make(map[string]interface{})
		for _, h := range vals {
			m, n := split(h, "=")
			mp[m] = n
		}

		data[k] = mp
	}

	return data
}

func split(s string, sp string) (k string, v string) {
	i := strings.Index(s, sp)
	if i == -1 {
		return s, ""
	}
	return s[:i], s[i+1:]
}
