package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/micro/cli/v2"
	"github.com/micro/go-micro/v2/client"
	gcli "github.com/micro/go-micro/v2/client/grpc"
	"github.com/micro/go-micro/v2/config/cmd"
	proto "github.com/micro/go-micro/v2/debug/service/proto"
	"github.com/micro/go-micro/v2/util/log"
	_ "github.com/micro/go-plugins/registry/kubernetes/v2"
)

const (
	// StatusInvalidArguments indicates specified invalid arguments.
	StatusInvalidArguments = 1
	// StatusConnectionFailure indicates connection failed.
	StatusConnectionFailure = 2
	// StatusUnhealthy indicates rpc succeeded but indicates unhealthy service.
	StatusUnhealthy = 4
)

var (
	serverAddress string
	serverName    string
	connTimeout   time.Duration
	rpcTimeout    time.Duration
	verbose       bool
)

func init() {
	os.Setenv("MICRO_REGISTRY", "kubernetes")
	client.DefaultClient = gcli.NewClient()

	argError := func(s string, v ...interface{}) {
		log.Logf("error: "+s, v...)
		os.Exit(StatusInvalidArguments)
	}

	app := cmd.App()
	app.Flags = append(app.Flags,
		&cli.DurationFlag{
			Name:        "connect_timeout",
			Value:       time.Second,
			Usage:       "timeout for establishing connection",
			EnvVars:     []string{"MICRO_CONNECT_TIMEOUT"},
			Destination: &connTimeout,
		},
		&cli.DurationFlag{
			Name:        "rpc_timeout",
			Value:       time.Second,
			Usage:       "timeout for health check rpc",
			EnvVars:     []string{"MICRO_RPC_TIMEOUT"},
			Destination: &rpcTimeout,
		},
		&cli.BoolFlag{
			Name:        "verbose",
			Usage:       "verbose logs",
			EnvVars:     []string{"MICRO_VERBOSE"},
			Destination: &verbose,
		},
	)

	app.Action = func(c *cli.Context) error {
		serverName = c.String("server_name")
		serverAddress = c.String("server_address")

		if len(serverName) == 0 {
			argError("server name not set")
		}
		if len(serverAddress) == 0 {
			argError("server address not set")
		}
		if connTimeout <= 0 {
			argError("connection timeout must be greater than zero (specified: %v)", connTimeout)
		}
		if rpcTimeout <= 0 {
			argError("rpc timeout must be greater than zero (specified: %v)", rpcTimeout)
		}
		return nil
	}

	cmd.Init()
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		sig := <-c
		if sig == os.Interrupt {
			log.Log("cancellation received")
			cancel()
			return
		}
	}()

	if !verbose {
		log.Log("establishing connection")
	}

	req := client.NewRequest(serverName, "Debug.Health", &proto.HealthRequest{})
	rsp := &proto.HealthResponse{}
	startTime := time.Now()

	err := client.Call(ctx, req, rsp, client.WithAddress(serverAddress), client.WithDialTimeout(connTimeout), client.WithRequestTimeout(connTimeout))
	if err != nil {
		if err == context.DeadlineExceeded {
			log.Logf("timeout: failed to connect service %q within %v", serverAddress, connTimeout)
		} else {
			log.Logf("error: failed to connect service at %q: %+v", serverAddress, err)
		}
		os.Exit(StatusConnectionFailure)
	}

	if !verbose {
		log.Logf("time elapsed: %v", time.Since(startTime))
	}

	if rsp.Status != "ok" {
		log.Logf("service unhealthy (responded with %q)", rsp.Status)
		os.Exit(StatusUnhealthy)
	}

	log.Logf("status: %v", rsp.Status)
}
