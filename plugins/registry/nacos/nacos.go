package nacos


import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/asim/go-micro/v3/cmd"
	"github.com/asim/go-micro/v3/registry"
	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

type nacosRegistry struct {
	client naming_client.INamingClient
	opts   registry.Options
}

func init() {
	cmd.DefaultRegistries["nacos"] = NewRegistry
}

// NewRegistry NewRegistry
func NewRegistry(opts ...registry.Option) registry.Registry {
	n := &nacosRegistry{
		opts: registry.Options{},
	}
	if err := configure(n, opts...); err != nil {
		panic(err)
	}
	return n
}

func configure(n *nacosRegistry, opts ...registry.Option) error {
	// set opts
	for _, o := range opts {
		o(&n.opts)
	}

	clientConfig := constant.ClientConfig{}
	serverConfigs := make([]constant.ServerConfig, 0)
	contextPath := "/nacos"

	cfg, ok := n.opts.Context.Value(configKey{}).(constant.ClientConfig)
	if ok {
		clientConfig = cfg
	}
	addrs, ok := n.opts.Context.Value(addressKey{}).([]string)
	if !ok {
		addrs = []string{"127.0.0.1:8848"} // 默认连接本地
	}

	for _, addr := range addrs {
		// check we have a port
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return err
		}

		p, err := strconv.ParseUint(port, 10, 64)
		if err != nil {
			return err
		}

		serverConfigs = append(serverConfigs, constant.ServerConfig{
			// Scheme:      "go.micro",
			IpAddr:      host,
			Port:        p,
			ContextPath: contextPath,
		})
	}

	if n.opts.Timeout == 0 {
		n.opts.Timeout = time.Second * 1
	}

	clientConfig.TimeoutMs = uint64(n.opts.Timeout.Milliseconds())
	client, err := clients.CreateNamingClient(map[string]interface{}{
		constant.KEY_SERVER_CONFIGS: serverConfigs,
		constant.KEY_CLIENT_CONFIG:  clientConfig,
	})
	if err != nil {
		return err
	}
	n.client = client

	return nil
}

func getNodeIPPort(s *registry.Service) (host string, port int, err error) {
	if len(s.Nodes) == 0 {
		return "", 0, errors.New("you must deregister at least one node")
	}
	node := s.Nodes[0]
	host, pt, err := net.SplitHostPort(node.Address)
	if err != nil {
		return "", 0, err
	}
	port, err = strconv.Atoi(pt)
	if err != nil {
		return "", 0, err
	}
	return
}

func (n *nacosRegistry) Init(opts ...registry.Option) error {
	_ = configure(n, opts...)
	return nil
}

func (n *nacosRegistry) Options() registry.Options {
	return n.opts
}

func (n *nacosRegistry) Register(s *registry.Service, opts ...registry.RegisterOption) error {
	var options registry.RegisterOptions
	for _, o := range opts {
		o(&options)
	}
	withContext := false
	param := vo.RegisterInstanceParam{}
	if options.Context != nil {
		if p, ok := options.Context.Value("register_instance_param").(vo.RegisterInstanceParam); ok {
			param = p
			withContext = ok
		}
	}
	if !withContext {
		host, port, err := getNodeIPPort(s)
		if err != nil {
			return err
		}
		s.Nodes[0].Metadata["version"] = s.Version
		param.Ip = host
		param.Port = uint64(port)
		param.Metadata = s.Nodes[0].Metadata
		param.ServiceName = s.Name
		param.Enable = true
		param.Healthy = true
		param.Weight = 1.0
		param.Ephemeral = true
	}
	_, err := n.client.RegisterInstance(param)
	logger.Info("test nacos logger")
	return err
}

func (n *nacosRegistry) Deregister(s *registry.Service, opts ...registry.DeregisterOption) error {
	var options registry.DeregisterOptions
	for _, o := range opts {
		o(&options)
	}
	withContext := false
	param := vo.DeregisterInstanceParam{}
	if options.Context != nil {
		if p, ok := options.Context.Value("deregister_instance_param").(vo.DeregisterInstanceParam); ok {
			param = p
			withContext = ok
		}
	}
	if !withContext {
		host, port, err := getNodeIPPort(s)
		if err != nil {
			return err
		}
		param.Ip = host
		param.Port = uint64(port)
		param.ServiceName = s.Name
	}

	_, err := n.client.DeregisterInstance(param)
	//log.Println(param, err)
	return err
}

func (n *nacosRegistry) GetService(name string, opts ...registry.GetOption) ([]*registry.Service, error) {
	var options registry.GetOptions
	for _, o := range opts {
		o(&options)
	}
	withContext := false
	param := vo.GetServiceParam{}
	if options.Context != nil {
		if p, ok := options.Context.Value("select_instances_param").(vo.GetServiceParam); ok {
			param = p
			withContext = ok
		}
	}
	if !withContext {
		param.ServiceName = name
	}
	service, err := n.client.GetService(param)
	if err != nil {
		return nil, err
	}
	services := make([]*registry.Service, 0)
	for _, v := range service.Hosts {
		//log.Printf("%+v\n", v)
		// 跳过不正常的节点
		if !v.Healthy || !v.Enable || v.Weight <= 0 {
			continue
		}

		nodes := make([]*registry.Node, 0)
		nodes = append(nodes, &registry.Node{
			Id:       v.InstanceId,
			Address:  net.JoinHostPort(v.Ip, fmt.Sprintf("%d", v.Port)),
			Metadata: v.Metadata,
		})
		s := registry.Service{
			Name:     v.ServiceName,
			Version:  v.Metadata["version"],
			Metadata: v.Metadata,
			Nodes:    nodes,
		}
		services = append(services, &s)
	}

	return services, nil
}

func (n *nacosRegistry) ListServices(opts ...registry.ListOption) ([]*registry.Service, error) {
	var options registry.ListOptions
	for _, o := range opts {
		o(&options)
	}
	withContext := false
	param := vo.GetAllServiceInfoParam{}
	if options.Context != nil {
		if p, ok := options.Context.Value("get_all_service_info_param").(vo.GetAllServiceInfoParam); ok {
			param = p
			withContext = ok
		}
	}
	if !withContext {
		services, err := n.client.GetAllServicesInfo(param)
		if err != nil {
			return nil, err
		}
		param.PageNo = 1
		param.PageSize = uint32(services.Count)
	}
	services, err := n.client.GetAllServicesInfo(param)
	if err != nil {
		return nil, err
	}
	var registryServices []*registry.Service
	for _, v := range services.Doms {
		registryServices = append(registryServices, &registry.Service{Name: v})
	}
	return registryServices, nil
}

func (n *nacosRegistry) Watch(opts ...registry.WatchOption) (registry.Watcher, error) {
	return newWatcher(n, opts...)
}

func (n *nacosRegistry) String() string {
	return "nacos"
}
