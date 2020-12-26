package eureka

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hudl/fargo"
	"github.com/micro/go-micro/v2/registry"
)

func TestServiceToInstance(t *testing.T) {
	nodes := []*registry.Node{
		&registry.Node{
			Id:       "node0",
			Address:  "node0.example.com:1234",
			Metadata: map[string]string{"foo": "bar"},
		},
		&registry.Node{
			Id:      "node1",
			Address: "node1.example.com:9876",
		},
	}

	endpoints := []*registry.Endpoint{
		&registry.Endpoint{
			Name:     "endpoint",
			Request:  &registry.Value{"request-value", "request-value-type", []*registry.Value{}},
			Response: &registry.Value{"response-value", "response-value-type", []*registry.Value{}},
			Metadata: map[string]string{"endpoint-meta-key": "endpoint-meta-value"},
		},
	}

	service := &registry.Service{
		Name:      "service-name",
		Version:   "service-version",
		Nodes:     nodes,
		Endpoints: endpoints,
	}

	instance, err := serviceToInstance(service)
	if err != nil {
		t.Error("Unexpected serviceToInstance error:", err)
	}

	instanceMetadata := instance.Metadata.GetMap()

	expectedUniqueID := fmt.Sprintf("%s:%s", nodes[0].Address, nodes[0].Id)

	expectedEndpointsJSON, err := json.Marshal(endpoints)
	if err != nil {
		t.Error("Unexpected endpoints marshal error:", err)
	}

	expectedNodeMetadataJSON, err := json.Marshal(nodes[0].Metadata)
	if err != nil {
		t.Error("Unexpected node metadata marshal error:", err)
	}

	testData := []struct {
		name string
		want interface{}
		got  interface{}
	}{
		{"instance.App", service.Name, instance.App},
		{"instance.HostName", nodes[0].Address, instance.HostName},
		{"instance.IPAddr", nodes[0].Address, instance.IPAddr},
		{"instance.VipAddress", nodes[0].Address, instance.VipAddress},
		{"instance.SecureVipAddress", nodes[0].Address, instance.SecureVipAddress},
		{"instance.Status", fargo.UP, instance.Status},
		{"instance.UniqueID()", expectedUniqueID, instance.UniqueID(*instance)},
		{"instance.DataCenteInfo.Name", fargo.MyOwn, instance.DataCenterInfo.Name},
		{`instance.Metadata["version"]`, service.Version, instanceMetadata["version"]},
		{`instance.Metadata["instanceId"]`, nodes[0].Id, instanceMetadata["instanceId"]},
		{`instance.Metadata["endpoints"]`, string(expectedEndpointsJSON), instanceMetadata["endpoints"]},
		{`instance.Metadata["metadata"]`, string(expectedNodeMetadataJSON), instanceMetadata["metadata"]},
	}

	for _, test := range testData {
		if test.got != test.want {
			t.Errorf("Unexpected %s: want %v, got %v", test.name, test.want, test.got)
		}
	}
}
