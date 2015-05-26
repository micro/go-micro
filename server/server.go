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
	Config() options
	Init(...Option)
	NewReceiver(interface{}) Receiver
	NewNamedReceiver(string, interface{}) Receiver
	Register(Receiver) error
	Start() error
	Stop() error
}

type Option func(*options)

var (
	DefaultAddress        = ":0"
	DefaultName           = "go-server"
	DefaultId             = uuid.NewUUID().String()
	DefaultServer  Server = newRpcServer()
)

func Name(n string) Option {
	return func(o *options) {
		o.name = n
	}
}

func Id(id string) Option {
	return func(o *options) {
		o.id = id
	}
}

func Address(a string) Option {
	return func(o *options) {
		o.address = a
	}
}

func Transport(t transport.Transport) Option {
	return func(o *options) {
		o.transport = t
	}
}

func Metadata(md map[string]string) Option {
	return func(o *options) {
		o.metadata = md
	}
}

func Config() options {
	return DefaultServer.Config()
}

func Init(opt ...Option) {
	if DefaultServer == nil {
		DefaultServer = newRpcServer(opt...)
	}
	DefaultServer.Init(opt...)
}

func NewServer(opt ...Option) Server {
	return newRpcServer(opt...)
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
	config := DefaultServer.Config()
	var host string
	var port int
	parts := strings.Split(config.Address(), ":")
	if len(parts) > 1 {
		host = strings.Join(parts[:len(parts)-1], ":")
		port, _ = strconv.Atoi(parts[len(parts)-1])
	} else {
		host = parts[0]
	}

	// register service
	node := &registry.Node{
		Id:       config.Id(),
		Address:  host,
		Port:     port,
		Metadata: config.Metadata(),
	}

	service := &registry.Service{
		Name:  config.Name(),
		Nodes: []*registry.Node{node},
	}

	log.Infof("Registering node: %s", node.Id)

	err := registry.Register(service)
	if err != nil {
		log.Fatal("Failed to register: %v", err)
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL)
	log.Infof("Received signal %s", <-ch)

	log.Infof("Deregistering %s", node.Id)
	registry.Deregister(service)

	return Stop()
}

func Start() error {
	config := DefaultServer.Config()
	log.Infof("Starting server %s id %s", config.Name(), config.Id())
	return DefaultServer.Start()
}

func Stop() error {
	log.Infof("Stopping server")
	return DefaultServer.Stop()
}
