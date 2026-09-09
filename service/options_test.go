package service

import "testing"

func TestLocalOption(t *testing.T) {
	// Off by default.
	if newOptions().Client.Options().Local {
		t.Fatal("Local should be off by default")
	}
	// Enabled by the option, on the service's own client.
	if !newOptions(Local()).Client.Options().Local {
		t.Fatal("Local() did not enable the client fast-path")
	}
}
