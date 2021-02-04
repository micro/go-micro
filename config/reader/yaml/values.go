package yaml

import (
	"fmt"
	"github.com/asim/go-micro/v3/config/reader"
	"github.com/asim/go-micro/v3/config/source"
	"github.com/ghodss/yaml"
	simple "gitlab.com/zzjin/go-simpleyaml"
	"strconv"
	"strings"
	"time"
)

type yamlValues struct {
	ch *source.ChangeSet
	sy *simple.Yaml
}

func newValues(ch *source.ChangeSet) (reader.Values, error) {
	sy := simple.New()
	data, _ := reader.ReplaceEnvVars(ch.Data)
	if err := sy.Unmarshal(data); err != nil {
		sy.SetPath(nil, string(ch.Data))
	}
	return &yamlValues{ch, sy}, nil
}

func (y *yamlValues) Bytes() []byte {
	b, _ := y.sy.Marshal()
	return b
}

func (y *yamlValues) Get(path ...string) reader.Value {
	return &yamlValue{y.sy.GetPath(path...)}
}

func (y *yamlValues) Set(val interface{}, path ...string) {
	y.sy.SetPath(path, val)
}

func (y *yamlValues) Del(path ...string) {
	if len(path) == 0 {
		y.sy = simple.New()
		return
	}

	if len(path) == 1 {
		y.sy.Del(path[0])
		return
	}

	vals := y.sy.GetPath(path[:len(path)-1]...)
	vals.Del(path[len(path)-1])
	y.sy.SetPath(path[:len(path)-1], vals.Interface())

	return
}

func (y *yamlValues) Map() map[string]interface{} {
	m, err := y.sy.Map()
	res := map[string]interface{}{}
	if err != nil {
		return res
	}
	for k, v := range m {
		sk := fmt.Sprintf("%v", k)
		res[sk] = v
	}

	return res
}

func (y *yamlValues) Scan(v interface{}) error {
	b, err := y.sy.Marshal()
	if err != nil {
		return err
	}

	return yaml.Unmarshal(b, v)
}

type yamlValue struct {
	*simple.Yaml
}

func (y *yamlValue) Bool(def bool) bool {
	b, err := y.Yaml.Bool()
	if err == nil {
		return b
	}

	s, ok := y.Interface().(string)
	if !ok {
		return def
	}

	b, err = strconv.ParseBool(s)
	if err != nil {
		return def
	}

	return b
}

func (y *yamlValue) Int(def int) int {
	i, err := y.Yaml.Int()
	if err == nil {
		return i
	}

	s, ok := y.Yaml.Interface().(string)
	if !ok {
		return def
	}

	i, err = strconv.Atoi(s)
	if err != nil {
		return def
	}

	return i
}

func (y *yamlValue) String(def string) string {
	return y.Yaml.MustString(def)
}

func (y *yamlValue) Float64(def float64) float64 {
	f, err := y.Yaml.Float64()
	if err == nil {
		return f
	}

	s, ok := y.Yaml.Interface().(string)
	if !ok {
		return def
	}

	f, err = strconv.ParseFloat(s, 64)
	if err != nil {
		return def
	}

	return f
}

func (y *yamlValue) Duration(def time.Duration) time.Duration {
	str, err := y.Yaml.String()
	if err != nil {
		return def
	}

	duration, err := time.ParseDuration(str)
	if err != nil {
		return def
	}

	return duration
}

func (y *yamlValue) StringSlice(def []string) []string {
	s, err := y.Yaml.String()
	if err != nil {
		sl := strings.Split(s, ",")
		if len(sl) > 1 {
			return sl
		}
	}

	return y.Yaml.MustStringArray(def)
}

func (y *yamlValue) StringMap(def map[string]string) map[string]string {
	m, err := y.Yaml.Map()
	if err != nil {
		return def
	}

	res := map[string]string{}
	for k, v := range m {
		sk, ok := k.(string)
		if !ok {
			return def
		}
		res[sk] = fmt.Sprintf("%v", v)
	}

	return res
}

func (y *yamlValue) Scan(val interface{}) error {
	b, err := y.Yaml.Marshal()
	if err != nil {
		return err
	}

	return yaml.Unmarshal(b, val)
}

func (y *yamlValue) Bytes() []byte {
	b, err := y.Yaml.Bytes()
	if err != nil {
		b, _ = y.Yaml.Marshal()
	}

	return b
}
