// Package defaults links every in-repo plugin into the cmd flag maps.
//
// Blank-import it to make all command-line plugin selections work, exactly as
// they did when cmd imported the plugins directly:
//
//	import _ "go-micro.dev/v6/cmd/defaults"
//
// The micro CLI imports it, so `micro --registry etcd ...` and friends behave
// unchanged. Library binaries that construct their own Registry, Broker,
// Store, and Transport can skip this import and shed the NATS, Consul, etcd,
// RabbitMQ, Redis, Postgres, MySQL, mDNS, and file (bbolt) machinery — roughly
// 40 packages — from their builds. A binary that skips it but still selects a
// plugin by flag gets a clear "not registered" error naming this package.
//
// Linking the defaults also restores the historical package defaults that
// used to be compiled into core: mdns discovery and file-backed storage
// (registry.DefaultRegistry, store.DefaultStore / store.NewStore). Core
// keeps lightweight in-memory defaults so binaries that link no plugins still
// work out of the box.
package defaults

import (
	"go-micro.dev/v6/cmd"

	nbroker "go-micro.dev/v6/broker/nats"
	rabbit "go-micro.dev/v6/broker/rabbitmq"
	"go-micro.dev/v6/cache/redis"
	"go-micro.dev/v6/client"
	"go-micro.dev/v6/registry/consul"
	"go-micro.dev/v6/registry/etcd"
	mdns "go-micro.dev/v6/registry/mdns"
	nregistry "go-micro.dev/v6/registry/nats"
	"go-micro.dev/v6/store/file"
	"go-micro.dev/v6/store/mysql"
	natsjskv "go-micro.dev/v6/store/nats-js-kv"
	postgres "go-micro.dev/v6/store/postgres"
	ntransport "go-micro.dev/v6/transport/nats"

	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/store"

	// Registers the "nats" plugin profile (--profile nats).
	_ "go-micro.dev/v6/service/profile/natsprofile"
)

func init() {
	cmd.DefaultBrokers["nats"] = nbroker.NewNatsBroker
	cmd.DefaultBrokers["rabbitmq"] = rabbit.NewBroker

	cmd.DefaultRegistries["consul"] = consul.NewConsulRegistry
	cmd.DefaultRegistries["etcd"] = etcd.NewEtcdRegistry
	cmd.DefaultRegistries["mdns"] = mdns.NewRegistry
	cmd.DefaultRegistries["nats"] = nregistry.NewNatsRegistry

	cmd.DefaultTransports["nats"] = ntransport.NewTransport

	cmd.DefaultStores["file"] = file.NewStore
	cmd.DefaultStores["mysql"] = mysql.NewMysqlStore
	cmd.DefaultStores["natsjskv"] = natsjskv.NewStore
	cmd.DefaultStores["postgres"] = postgres.NewStore

	cmd.DefaultCaches["redis"] = redis.NewRedisCache

	// Restore the historical defaults that core used to compile in: mdns
	// discovery and file-backed storage. Without this, a binary linking the
	// plugins (the CLI) would land on the lightweight memory defaults.
	registry.DefaultRegistry = mdns.NewRegistry()
	store.RegisterDefault(file.NewStore)

	// cmd.DefaultCmd's registry is snapshotted at package init, before the
	// plugins above link in, so point its snapshot at the restored default
	// too — cmd.Before echoes it back to registry.DefaultRegistry after flag
	// parsing and would otherwise clobber the freshly installed one.
	if r := cmd.DefaultCmd.Options().Registry; r != nil {
		*r = registry.DefaultRegistry
	}

	// client.DefaultClient snapshots registry.DefaultRegistry at package
	// init, before this package links in, so a CLI that calls out through
	// it (micro chat, api, gateway, flow, run) would resolve services
	// against the memory registry while everything else talks to mdns.
	// Rebuild it against the restored default.
	client.DefaultClient = client.NewClient(client.Registry(registry.DefaultRegistry))
}
