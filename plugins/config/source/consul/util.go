package consul

import (
	"fmt"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/micro/go-micro/v2/config/encoder"
)

type configValue interface {
	Value() interface{}
	Decode(encoder.Encoder, []byte) error
}
type configArrayValue struct {
	v []interface{}
}

func (a *configArrayValue) Value() interface{} { return a.v }
func (a *configArrayValue) Decode(e encoder.Encoder, b []byte) error {
	return e.Decode(b, &a.v)
}

type configMapValue struct {
	v map[string]interface{}
}

func (m *configMapValue) Value() interface{} { return m.v }
func (m *configMapValue) Decode(e encoder.Encoder, b []byte) error {
	return e.Decode(b, &m.v)
}

func makeMap(e encoder.Encoder, kv api.KVPairs, stripPrefix string) (map[string]interface{}, error) {

	data := make(map[string]interface{})

	// consul guarantees lexicographic order, so no need to sort
	for _, v := range kv {
		pathString := strings.TrimPrefix(strings.TrimPrefix(v.Key, strings.TrimPrefix(stripPrefix, "/")), "/")
		if pathString == "" {
			continue
		}
		var val configValue
		var err error

		// ensure a valid value is stored at this location
		if len(v.Value) > 0 {
			// try to decode into map value or array value
			arrayV := &configArrayValue{v: []interface{}{}}
			mapV := &configMapValue{v: map[string]interface{}{}}
			switch {
			case arrayV.Decode(e, v.Value) == nil:
				val = arrayV
			case mapV.Decode(e, v.Value) == nil:
				val = mapV
			default:
				return nil, fmt.Errorf("faild decode value. path: %s, error: %s", pathString, err)
			}
		}

		// set target at the root
		target := data
		path := strings.Split(pathString, "/")
		// find (or create) the leaf node we want to put this value at
		for _, dir := range path[:len(path)-1] {
			if _, ok := target[dir]; !ok {
				target[dir] = make(map[string]interface{})
			}
			target = target[dir].(map[string]interface{})
		}

		leafDir := path[len(path)-1]

		// copy over the keys from the value
		switch val.(type) {
		case *configArrayValue:
			target[leafDir] = val.Value()
		case *configMapValue:
			target[leafDir] = make(map[string]interface{})
			target = target[leafDir].(map[string]interface{})
			mapv := val.Value().(map[string]interface{})
			for k := range mapv {
				target[k] = mapv[k]
			}
		}
	}

	return data, nil
}
