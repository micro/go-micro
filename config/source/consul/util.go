package consul

import (
	"fmt"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/micro/go-micro/config/encoder"
)

func makeMap(e encoder.Encoder, kv api.KVPairs, stripPrefix string) (map[string]interface{}, error) {
	data := make(map[string]interface{})

	// consul guarantees lexicographic order, so no need to sort
	for _, v := range kv {
		pathString := strings.TrimPrefix(strings.TrimPrefix(v.Key, strings.TrimPrefix(stripPrefix, "/")), "/")
		var val map[string]interface{}

		// ensure a valid value is stored at this location
		if len(v.Value) > 0 {
			if err := e.Decode(v.Value, &val); err != nil {
				return nil, fmt.Errorf("faild decode value. path: %s, error: %s", pathString, err)
			}
		}

		// set target at the root
		target := data

		// then descend to the target location, creating as we go, if need be
		if pathString != "" {
			path := strings.Split(pathString, "/")
			// find (or create) the location we want to put this value at
			for _, dir := range path {
				if _, ok := target[dir]; !ok {
					target[dir] = make(map[string]interface{})
				}
				target = target[dir].(map[string]interface{})
			}

		}

		// copy over the keys from the value
		for k := range val {
			target[k] = val[k]
		}
	}

	return data, nil
}
