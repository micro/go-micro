package consul

import (
	"fmt"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/micro/go-micro/config/encoder"
)

type jsonValue interface {
	Value() interface{}
	Decode(encoder.Encoder, []byte) (jsonValue, error)
}
type jsonArrayValue []interface{}
type jsonMapValue map[string]interface{}

func (a jsonArrayValue) Value() interface{} { return a }
func (a jsonArrayValue) Decode(e encoder.Encoder, b []byte) (jsonValue, error) {
	v := jsonArrayValue{}
	err := e.Decode(b, &v)
	return v, err
}
func (m jsonMapValue) Value() interface{} { return m }
func (m jsonMapValue) Decode(e encoder.Encoder, b []byte) (jsonValue, error) {
	v := jsonMapValue{}
	err := e.Decode(b, &v)
	return v, err
}

func makeMap(e encoder.Encoder, kv api.KVPairs, stripPrefix string) (map[string]interface{}, error) {

	data := make(map[string]interface{})

	// consul guarantees lexicographic order, so no need to sort
	for _, v := range kv {
		pathString := strings.TrimPrefix(strings.TrimPrefix(v.Key, stripPrefix), "/")
		if pathString == "" {
			continue
		}
		var val jsonValue
		var err error

		// ensure a valid value is stored at this location
		if len(v.Value) > 0 {
			// check whether this is an array
			if v.Value[0] == 91 && v.Value[len(v.Value)-1] == 93 {
				val = jsonArrayValue{}
				if val, err = val.Decode(e, v.Value); err != nil {
					return nil, fmt.Errorf("faild decode value. path: %s, error: %s", pathString, err)
				}
			} else {
				val = jsonMapValue{}
				if val, err = val.Decode(e, v.Value); err != nil {
					return nil, fmt.Errorf("faild decode value. path: %s, error: %s", pathString, err)
				}
			}
		}

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
		case jsonArrayValue:
			target[leafDir] = val.Value()
		case jsonMapValue:
			target[leafDir] = make(map[string]interface{})
			target = target[leafDir].(map[string]interface{})
			mapv := val.Value().(jsonMapValue)
			for k := range mapv {
				target[k] = mapv[k]
			}
		}
	}

	return data, nil
}
