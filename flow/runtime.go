package flow

import (
	"reflect"
	"runtime"
	"strings"
)

// Get Serevice endpoint from provided interface
func ServiceEndpoint(iface interface{}) string {
	svc := runtime.FuncForPC(reflect.ValueOf(iface).Pointer()).Name()
	idx1 := strings.LastIndex(svc, ".")
	idx2 := strings.LastIndex(svc[:idx1], ".")
	return svc[idx2+1:]
}
