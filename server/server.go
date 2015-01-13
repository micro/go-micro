package server

import (
	"flag"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"code.google.com/p/go-uuid/uuid"
	"github.com/asim/go-micro/registry"
	"github.com/asim/go-micro/store"
	log "github.com/cihub/seelog"
)

type Server interface {
	Address() string
	Init() error
	NewReceiver(interface{}) Receiver
	NewNamedReceiver(string, interface{}) Receiver
	Register(Receiver) error
	Start() error
	Stop() error
}

var (
	Name          string
	Id            string
	DefaultServer Server

	flagRegistry    string
	flagBindAddress string
)

func init() {
	flag.StringVar(&flagRegistry, "registry", "consul", "Registry for discovery. kubernetes, consul, etc")
	flag.StringVar(&flagBindAddress, "bind_address", ":0", "Bind address for the server. 127.0.0.1:8080")
}

func Init() error {
	flag.Parse()

	switch flagRegistry {
	case "kubernetes":
		registry.DefaultRegistry = registry.NewKubernetesRegistry()
		store.DefaultStore = store.NewMemcacheStore()
	}

	if len(Name) == 0 {
		Name = "go-server"
	}

	if len(Id) == 0 {
		Id = Name + "-" + uuid.NewUUID().String()
	}

	if DefaultServer == nil {
		DefaultServer = NewRpcServer(flagBindAddress)
	}

	return DefaultServer.Init()
}

func NewReceiver(handler interface{}) Receiver {
	return DefaultServer.NewReceiver(handler)
}

func NewNamedReceiver(path string, handler interface{}) Receiver {
	return DefaultServer.NewNamedReceiver(path, handler)
}

func Register(r Receiver) error {
	return DefaultServer.Register(r)
}

func Run() error {
	if err := Start(); err != nil {
		return err
	}

	// parse address for host, port
	parts := strings.Split(DefaultServer.Address(), ":")
	host := strings.Join(parts[:len(parts)-1], ":")
	port, _ := strconv.Atoi(parts[len(parts)-1])

	// register service
	node := registry.NewNode(Id, host, port)
	service := registry.NewService(Name, node)

	log.Debugf("Registering %s", node.Id())
	registry.Register(service)

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL)
	log.Debugf("Received signal %s", <-ch)

	log.Debugf("Deregistering %s", node.Id())
	registry.Deregister(service)

	return Stop()
}

func Start() error {
	log.Debugf("Starting server %s id %s", Name, Id)
	return DefaultServer.Start()
}

func Stop() error {
	log.Debugf("Stopping server")
	return DefaultServer.Stop()
}
