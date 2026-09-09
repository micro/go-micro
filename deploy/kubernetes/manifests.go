package kubernetes

import (
	"embed"
	"fmt"
)

// crdFS holds the canonical CRD manifests. They live as real YAML under
// config/crd/ so they can be applied directly (`kubectl apply -f
// deploy/kubernetes/config/crd/`) and are embedded here so the Go API serves
// the exact same bytes — one source of truth, no drift.
//
//go:embed config/crd/agent.yaml config/crd/service.yaml config/crd/flow.yaml
var crdFS embed.FS

// CRDManifests contains the alpha CRDs for Go Micro lifecycle resources, loaded
// from the embedded config/crd/ YAML.
var CRDManifests = map[Kind]string{
	KindAgent:   mustCRD("agent"),
	KindService: mustCRD("service"),
	KindFlow:    mustCRD("flow"),
}

// mustCRD reads an embedded CRD manifest. The files are embedded at compile
// time, so a read error means a build/packaging bug, not a runtime condition.
func mustCRD(name string) string {
	b, err := crdFS.ReadFile("config/crd/" + name + ".yaml")
	if err != nil {
		panic(fmt.Sprintf("kubernetes: embedded CRD %q missing: %v", name, err))
	}
	return string(b)
}
