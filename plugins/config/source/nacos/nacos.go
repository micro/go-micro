package nacos

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/asim/go-micro/v3/config/source"
	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

type nacosConfigSource struct {
	confClient config_client.IConfigClient
	opts       source.Options
	group      string
	dataId     string
}

func NewSource(opts ...source.Option) source.Source {
	s := &nacosConfigSource{
		opts: source.Options{},
	}

	_ = sourceConfiguration(s, opts...)

	return s
}

func sourceConfiguration(s *nacosConfigSource, opts ...source.Option) error {
	s.opts = source.NewOptions(opts...)

	clientConfig := constant.ClientConfig{}
	serverConfigs := make([]constant.ServerConfig, 0)
	contextPath := "/nacos"

	g, ok := s.opts.Context.Value(groupKey{}).(string)
	if !ok {
		return fmt.Errorf("group must be setted")
	}
	s.group = g

	id, ok := s.opts.Context.Value(dataIdKey{}).(string)
	if !ok {
		return fmt.Errorf("dataId must be setted")
	}
	s.dataId = id

	cfg, ok := s.opts.Context.Value(configKey{}).(constant.ClientConfig)
	if ok {
		clientConfig = cfg
	}
	addrs, ok := s.opts.Context.Value(addressKey{}).([]string)
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

	ic, err := clients.NewConfigClient(vo.NacosClientParam{
		ClientConfig:  &clientConfig,
		ServerConfigs: serverConfigs,
	})
	if err != nil {
		return err
	}

	s.confClient = ic
	return nil
}

func (n *nacosConfigSource) Read() (*source.ChangeSet, error) {
	str, err := n.confClient.GetConfig(vo.ConfigParam{
		DataId: n.dataId,
		Group:  n.group,
	})
	if err != nil {
		return nil, err
	}

	cs := &source.ChangeSet{
		Timestamp: time.Now(),
		Format:    n.opts.Encoder.String(),
		Source:    n.String(),
		Data:      []byte(str),
	}
	cs.Checksum = cs.Sum()

	return cs, nil
}

func (n *nacosConfigSource) Write(set *source.ChangeSet) error {
	return nil
}

func (n *nacosConfigSource) Watch() (source.Watcher, error) {
	return newConfigWatcher(n.confClient, n.opts.Encoder, n.String(), n.group, n.dataId)
}

func (n *nacosConfigSource) String() string {
	return "nacos"
}
