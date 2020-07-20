package micro

import (
	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/cmd"
	"github.com/micro/go-micro/v2/config"
	"github.com/micro/go-micro/v2/debug/profile/http"
	"github.com/micro/go-micro/v2/debug/profile/pprof"
	"github.com/micro/go-micro/v2/debug/trace"
	"github.com/micro/go-micro/v2/server"
	"github.com/micro/go-micro/v2/store"

	// clients
	gcli "github.com/micro/go-micro/v2/client/grpc"
	cmucp "github.com/micro/go-micro/v2/client/mucp"

	gsrv "github.com/micro/go-micro/v2/server/grpc"
	smucp "github.com/micro/go-micro/v2/server/mucp"

	// brokers
	brokerHttp "github.com/micro/go-micro/v2/broker/http"
	"github.com/micro/go-micro/v2/broker/memory"
	"github.com/micro/go-micro/v2/broker/nats"
	brokerSrv "github.com/micro/go-micro/v2/broker/service"

	// registries
	"github.com/micro/go-micro/v2/registry/etcd"
	"github.com/micro/go-micro/v2/registry/mdns"
	rmem "github.com/micro/go-micro/v2/registry/memory"
	regSrv "github.com/micro/go-micro/v2/registry/service"

	// routers
	dnsRouter "github.com/micro/go-micro/v2/router/dns"
	regRouter "github.com/micro/go-micro/v2/router/registry"
	srvRouter "github.com/micro/go-micro/v2/router/service"
	staticRouter "github.com/micro/go-micro/v2/router/static"

	// runtimes
	kRuntime "github.com/micro/go-micro/v2/runtime/kubernetes"
	lRuntime "github.com/micro/go-micro/v2/runtime/local"
	srvRuntime "github.com/micro/go-micro/v2/runtime/service"

	// selectors
	randSelector "github.com/micro/go-micro/v2/selector/random"
	roundSelector "github.com/micro/go-micro/v2/selector/roundrobin"

	// transports
	thttp "github.com/micro/go-micro/v2/transport/http"
	tmem "github.com/micro/go-micro/v2/transport/memory"

	// stores
	memStore "github.com/micro/go-micro/v2/store/memory"
	svcStore "github.com/micro/go-micro/v2/store/service"

	// tracers
	// jTracer "github.com/micro/go-micro/v2/debug/trace/jaeger"
	memTracer "github.com/micro/go-micro/v2/debug/trace/memory"

	// auth
	jwtAuth "github.com/micro/go-micro/v2/auth/jwt"
	svcAuth "github.com/micro/go-micro/v2/auth/service"
)

func init() {
	// set defaults

	// default client
	client.DefaultClient = gcli.NewClient()
	// default server
	server.DefaultServer = gsrv.NewServer()
	// default store
	store.DefaultStore = memStore.NewStore()
	// set default trace
	trace.DefaultTracer = memTracer.NewTracer()

	// import all the plugins

	// auth
	cmd.DefaultAuths["service"] = svcAuth.NewAuth
	cmd.DefaultAuths["jwt"] = jwtAuth.NewAuth

	// broker
	cmd.DefaultBrokers["service"] = brokerSrv.NewBroker
	cmd.DefaultBrokers["memory"] = memory.NewBroker
	cmd.DefaultBrokers["nats"] = nats.NewBroker
	cmd.DefaultBrokers["http"] = brokerHttp.NewBroker

	// config
	cmd.DefaultConfigs["service"] = config.NewConfig

	// client
	cmd.DefaultClients["mucp"] = cmucp.NewClient
	cmd.DefaultClients["grpc"] = gcli.NewClient

	// profiler
	cmd.DefaultProfiles["http"] = http.NewProfile
	cmd.DefaultProfiles["pprof"] = pprof.NewProfile

	// registry
	cmd.DefaultRegistries["service"] = regSrv.NewRegistry
	cmd.DefaultRegistries["etcd"] = etcd.NewRegistry
	cmd.DefaultRegistries["mdns"] = mdns.NewRegistry
	cmd.DefaultRegistries["memory"] = rmem.NewRegistry

	// runtime
	cmd.DefaultRuntimes["local"] = lRuntime.NewRuntime
	cmd.DefaultRuntimes["service"] = srvRuntime.NewRuntime
	cmd.DefaultRuntimes["kubernetes"] = kRuntime.NewRuntime

	// router
	cmd.DefaultRouters["dns"] = dnsRouter.NewRouter
	cmd.DefaultRouters["registry"] = regRouter.NewRouter
	cmd.DefaultRouters["static"] = staticRouter.NewRouter
	cmd.DefaultRouters["service"] = srvRouter.NewRouter

	// selector
	cmd.DefaultSelectors["random"] = randSelector.NewSelector
	cmd.DefaultSelectors["roundrobin"] = roundSelector.NewSelector

	// server
	cmd.DefaultServers["mucp"] = smucp.NewServer
	cmd.DefaultServers["grpc"] = gsrv.NewServer

	// store
	cmd.DefaultStores["memory"] = memStore.NewStore
	cmd.DefaultStores["service"] = svcStore.NewStore

	// trace
	cmd.DefaultTracers["memory"] = memTracer.NewTracer

	// transport
	cmd.DefaultTransports["memory"] = tmem.NewTransport
	cmd.DefaultTransports["http"] = thttp.NewTransport
}
