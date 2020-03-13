package etcd

import (
	"context"
	cryptotls "crypto/tls"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/micro/go-micro/v2/store"
	"google.golang.org/grpc"
)

// Implement all the options from https://pkg.go.dev/go.etcd.io/etcd/clientv3?tab=doc#Config
// Need to use non basic types in context.WithValue
type autoSyncInterval string
type dialTimeout string
type dialKeepAliveTime string
type dialKeepAliveTimeout string
type maxCallSendMsgSize string
type maxCallRecvMsgSize string
type tls string
type username string
type password string
type rejectOldCluster string
type dialOptions string
type clientContext string
type permitWithoutStream string

// AutoSyncInterval is the interval to update endpoints with its latest members.
// 0 disables auto-sync. By default auto-sync is disabled.
func AutoSyncInterval(d time.Duration) store.Option {
	return func(o *store.Options) {
		o.Context = context.WithValue(o.Context, autoSyncInterval(""), d)
	}
}

// DialTimeout is the timeout for failing to establish a connection.
func DialTimeout(d time.Duration) store.Option {
	return func(o *store.Options) {
		o.Context = context.WithValue(o.Context, dialTimeout(""), d)
	}
}

// DialKeepAliveTime is the time after which client pings the server to see if
// transport is alive.
func DialKeepAliveTime(d time.Duration) store.Option {
	return func(o *store.Options) {
		o.Context = context.WithValue(o.Context, dialKeepAliveTime(""), d)
	}
}

// DialKeepAliveTimeout is the time that the client waits for a response for the
// keep-alive probe. If the response is not received in this time, the connection is closed.
func DialKeepAliveTimeout(d time.Duration) store.Option {
	return func(o *store.Options) {
		o.Context = context.WithValue(o.Context, dialKeepAliveTimeout(""), d)
	}
}

// MaxCallSendMsgSize is the client-side request send limit in bytes.
// If 0, it defaults to 2.0 MiB (2 * 1024 * 1024).
// Make sure that "MaxCallSendMsgSize" < server-side default send/recv limit.
// ("--max-request-bytes" flag to etcd or "embed.Config.MaxRequestBytes").
func MaxCallSendMsgSize(size int) store.Option {
	return func(o *store.Options) {
		o.Context = context.WithValue(o.Context, maxCallSendMsgSize(""), size)
	}
}

// MaxCallRecvMsgSize is the client-side response receive limit.
// If 0, it defaults to "math.MaxInt32", because range response can
// easily exceed request send limits.
// Make sure that "MaxCallRecvMsgSize" >= server-side default send/recv limit.
// ("--max-request-bytes" flag to etcd or "embed.Config.MaxRequestBytes").
func MaxCallRecvMsgSize(size int) store.Option {
	return func(o *store.Options) {
		o.Context = context.WithValue(o.Context, maxCallRecvMsgSize(""), size)
	}
}

// TLS holds the client secure credentials, if any.
func TLS(conf *cryptotls.Config) store.Option {
	return func(o *store.Options) {
		t := conf.Clone()
		o.Context = context.WithValue(o.Context, tls(""), t)
	}
}

// Username is a user name for authentication.
func Username(u string) store.Option {
	return func(o *store.Options) {
		o.Context = context.WithValue(o.Context, username(""), u)
	}
}

// Password is a password for authentication.
func Password(p string) store.Option {
	return func(o *store.Options) {
		o.Context = context.WithValue(o.Context, password(""), p)
	}
}

// RejectOldCluster when set will refuse to create a client against an outdated cluster.
func RejectOldCluster(b bool) store.Option {
	return func(o *store.Options) {
		o.Context = context.WithValue(o.Context, rejectOldCluster(""), b)
	}
}

// DialOptions is a list of dial options for the grpc client (e.g., for interceptors).
// For example, pass "grpc.WithBlock()" to block until the underlying connection is up.
// Without this, Dial returns immediately and connecting the server happens in background.
func DialOptions(opts []grpc.DialOption) store.Option {
	return func(o *store.Options) {
		if len(opts) > 0 {
			ops := make([]grpc.DialOption, len(opts))
			copy(ops, opts)
			o.Context = context.WithValue(o.Context, dialOptions(""), ops)
		}
	}
}

// ClientContext is the default etcd3 client context; it can be used to cancel grpc
// dial out andother operations that do not have an explicit context.
func ClientContext(ctx context.Context) store.Option {
	return func(o *store.Options) {
		o.Context = context.WithValue(o.Context, clientContext(""), ctx)
	}
}

// PermitWithoutStream when set will allow client to send keepalive pings to server without any active streams(RPCs).
func PermitWithoutStream(b bool) store.Option {
	return func(o *store.Options) {
		o.Context = context.WithValue(o.Context, permitWithoutStream(""), b)
	}
}

func (e *etcdStore) applyConfig(cfg *clientv3.Config) {
	if v := e.options.Context.Value(autoSyncInterval("")); v != nil {
		cfg.AutoSyncInterval = v.(time.Duration)
	}
	if v := e.options.Context.Value(dialTimeout("")); v != nil {
		cfg.DialTimeout = v.(time.Duration)
	}
	if v := e.options.Context.Value(dialKeepAliveTime("")); v != nil {
		cfg.DialKeepAliveTime = v.(time.Duration)
	}
	if v := e.options.Context.Value(dialKeepAliveTimeout("")); v != nil {
		cfg.DialKeepAliveTimeout = v.(time.Duration)
	}
	if v := e.options.Context.Value(maxCallSendMsgSize("")); v != nil {
		cfg.MaxCallSendMsgSize = v.(int)
	}
	if v := e.options.Context.Value(maxCallRecvMsgSize("")); v != nil {
		cfg.MaxCallRecvMsgSize = v.(int)
	}
	if v := e.options.Context.Value(tls("")); v != nil {
		cfg.TLS = v.(*cryptotls.Config)
	}
	if v := e.options.Context.Value(username("")); v != nil {
		cfg.Username = v.(string)
	}
	if v := e.options.Context.Value(password("")); v != nil {
		cfg.Username = v.(string)
	}
	if v := e.options.Context.Value(rejectOldCluster("")); v != nil {
		cfg.RejectOldCluster = v.(bool)
	}
	if v := e.options.Context.Value(dialOptions("")); v != nil {
		cfg.DialOptions = v.([]grpc.DialOption)
	}
	if v := e.options.Context.Value(clientContext("")); v != nil {
		cfg.Context = v.(context.Context)
	}
	if v := e.options.Context.Value(permitWithoutStream("")); v != nil {
		cfg.PermitWithoutStream = v.(bool)
	}
}
