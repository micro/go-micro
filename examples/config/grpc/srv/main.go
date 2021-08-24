package main

import (
	"context"
	"crypto/md5"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	yaml "github.com/asim/go-micro/plugins/config/encoder/yaml/v3"
	proto "github.com/asim/go-micro/plugins/config/source/grpc/v3/proto"
	"github.com/asim/go-micro/v3/config"
	"github.com/asim/go-micro/v3/config/reader"
	"github.com/asim/go-micro/v3/config/reader/json"
	"github.com/asim/go-micro/v3/config/source/file"
	log "github.com/asim/go-micro/v3/logger"
	grpc "google.golang.org/grpc"
)

var (
	mux        sync.RWMutex
	configMaps = make(map[string]*proto.ChangeSet)
	apps       = []string{"micro", "extra"}
	cfg        config.Config
)

type Service struct{}

func main() {
	// create config with yaml encoder
	enc := yaml.NewEncoder()
	cfg, _ = config.NewConfig(config.WithReader(json.NewReader(
		reader.WithEncoder(enc),
	)))

	// load config files
	err := loadConfigFile()
	if err != nil {
		log.Fatal(err)
	}

	// new service
	service := grpc.NewServer()
	proto.RegisterSourceServer(service, new(Service))
	ts, err := net.Listen("tcp", ":8600")
	if err != nil {
		log.Fatal(err)
	}

	log.Infof("configServer started")
	err = service.Serve(ts)
	if err != nil {
		log.Fatal(err)
	}
}

func (s Service) Read(ctx context.Context, req *proto.ReadRequest) (rsp *proto.ReadResponse, err error) {
	appName := parsePath(req.Path)
	switch appName {
	case "micro", "extra":
		rsp = &proto.ReadResponse{
			ChangeSet: getConfig(appName),
		}
		return
	default:
		err = fmt.Errorf("[Read] the first path is invalid")
		return
	}
}

func (s Service) Watch(req *proto.WatchRequest, server proto.Source_WatchServer) (err error) {
	appName := parsePath(req.Path)
	rsp := &proto.WatchResponse{
		ChangeSet: getConfig(appName),
	}
	if err = server.Send(rsp); err != nil {
		log.Infof("[Watch] watch files error，%s", err)
		return err
	}

	return
}

func loadConfigFile() (err error) {
	for _, app := range apps {
		if err := cfg.Load(file.NewSource(
			file.WithPath("./conf/" + app + ".yaml"),
		)); err != nil {
			log.Fatalf("[loadConfigFile] load files error，%s", err)
			return err
		}
	}

	// watch changes
	watcher, err := cfg.Watch()
	if err != nil {
		log.Fatalf("[loadConfigFile] start watching files error，%s", err)
		return err
	}

	go func() {
		for {
			v, err := watcher.Next()
			if err != nil {
				log.Fatalf("[loadConfigFile] watch files error，%s", err)
				return
			}

			log.Infof("[loadConfigFile] file change， %s", string(v.Bytes()))
		}
	}()

	return
}

func getConfig(appName string) *proto.ChangeSet {
	bytes := cfg.Get(appName).Bytes()

	log.Infof("[getConfig] appName，%s", string(bytes))
	return &proto.ChangeSet{
		Data:      bytes,
		Checksum:  fmt.Sprintf("%x", md5.Sum(bytes)),
		Format:    "yml",
		Source:    "file",
		Timestamp: time.Now().Unix()}
}

func parsePath(path string) (appName string) {
	paths := strings.Split(path, "/")

	if paths[0] == "" && len(paths) > 1 {
		return paths[1]
	}

	return paths[0]
}
