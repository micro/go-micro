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
// RabbitMQ, Redis, Postgres, and MySQL machinery — roughly 40 packages — from
// their builds. A binary that skips it but still selects a plugin by flag gets
// a clear "not registered" error naming this package.
package defaults

import (
	"go-micro.dev/v6/cmd"

	nbroker "go-micro.dev/v6/broker/nats"
	rabbit "go-micro.dev/v6/broker/rabbitmq"
	"go-micro.dev/v6/cache/redis"
	"go-micro.dev/v6/registry/consul"
	"go-micro.dev/v6/registry/etcd"
	nregistry "go-micro.dev/v6/registry/nats"
	"go-micro.dev/v6/store/mysql"
	natsjskv "go-micro.dev/v6/store/nats-js-kv"
	postgres "go-micro.dev/v6/store/postgres"
	ntransport "go-micro.dev/v6/transport/nats"

	// Registers the "nats" plugin profile (--profile nats).
	_ "go-micro.dev/v6/service/profile/natsprofile"
)

func init() {
	cmd.DefaultBrokers["nats"] = nbroker.NewNatsBroker
	cmd.DefaultBrokers["rabbitmq"] = rabbit.NewBroker

	cmd.DefaultRegistries["consul"] = consul.NewConsulRegistry
	cmd.DefaultRegistries["etcd"] = etcd.NewEtcdRegistry
	cmd.DefaultRegistries["nats"] = nregistry.NewNatsRegistry

	cmd.DefaultTransports["nats"] = ntransport.NewTransport

	cmd.DefaultStores["mysql"] = mysql.NewMysqlStore
	cmd.DefaultStores["natsjskv"] = natsjskv.NewStore
	cmd.DefaultStores["postgres"] = postgres.NewStore

	cmd.DefaultCaches["redis"] = redis.NewRedisCache
}
