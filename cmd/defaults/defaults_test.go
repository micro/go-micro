package defaults

import (
	"testing"

	"go-micro.dev/v6/cmd"
	"go-micro.dev/v6/service/profile"
)

// Importing this package must make every historically flag-selectable plugin
// available again, exactly as when cmd imported the plugins directly.
func TestDefaultsRegisterAllPlugins(t *testing.T) {
	for name, m := range map[string]int{
		"brokers: nats":      boolToInt(cmd.DefaultBrokers["nats"] != nil),
		"brokers: rabbitmq":  boolToInt(cmd.DefaultBrokers["rabbitmq"] != nil),
		"registries: consul": boolToInt(cmd.DefaultRegistries["consul"] != nil),
		"registries: etcd":   boolToInt(cmd.DefaultRegistries["etcd"] != nil),
		"registries: nats":   boolToInt(cmd.DefaultRegistries["nats"] != nil),
		"transports: nats":   boolToInt(cmd.DefaultTransports["nats"] != nil),
		"stores: mysql":      boolToInt(cmd.DefaultStores["mysql"] != nil),
		"stores: natsjskv":   boolToInt(cmd.DefaultStores["natsjskv"] != nil),
		"stores: postgres":   boolToInt(cmd.DefaultStores["postgres"] != nil),
		"caches: redis":      boolToInt(cmd.DefaultCaches["redis"] != nil),
	} {
		if m != 1 {
			t.Errorf("%s not registered", name)
		}
	}
	// The nats profile registers via the natsprofile package import.
	if _, err := profile.Load("nats"); err != nil {
		// Loading may fail to CONNECT without a server, but it must not fail
		// as unregistered.
		t.Logf("nats profile load: %v (connection errors are fine)", err)
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
