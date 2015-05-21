package server

import (
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"code.google.com/p/go-uuid/uuid"
	log "github.com/golang/glog"
	"github.com/myodc/go-micro/registry"
	"github.com/myodc/go-micro/transport"
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

type options struct {
	transport transport.Transport
}

type Option func(*options)

var (
	Address       string
	Name          string
	Id            string
	DefaultServer Server
)

func Transport(t transport.Transport) Option {
	return func(o *options) {
		o.transport = t
	}
}

func Init() error {
	defer log.Flush()

	if len(Name) == 0 {
		Name = "go-server"
	}

	if len(Id) == 0 {
		Id = Name + "-" + uuid.NewUUID().String()
	}

	if DefaultServer == nil {
		DefaultServer = NewRpcServer(Address)
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
	var host string
	var port int
	parts := strings.Split(DefaultServer.Address(), ":")
	if len(parts) > 1 {
		host = strings.Join(parts[:len(parts)-1], ":")
		port, _ = strconv.Atoi(parts[len(parts)-1])
	} else {
		host = parts[0]
	}

	// register service
	node := registry.NewNode(Id, host, port)
	service := registry.NewService(Name, node)

	log.Infof("Registering node: %s", node.Id())

	err := registry.Register(service)
	if err != nil {
		log.Fatal("Failed to register: %v", err)
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL)
	log.Infof("Received signal %s", <-ch)

	log.Infof("Deregistering %s", node.Id())
	registry.Deregister(service)

	return Stop()
}

func Start() error {
	log.Infof("Starting server %s id %s", Name, Id)
	return DefaultServer.Start()
}

func Stop() error {
	log.Infof("Stopping server")
	return DefaultServer.Stop()
}
