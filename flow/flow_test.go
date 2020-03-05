package flow_test

import (
	"context"
	"testing"
	"time"

	micro "github.com/micro/go-micro/v2"
	"github.com/micro/go-micro/v2/broker"
	mbroker "github.com/micro/go-micro/v2/broker/memory"
	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/client/selector"
	rselector "github.com/micro/go-micro/v2/client/selector/registry"
	"github.com/micro/go-micro/v2/flow"
	proto "github.com/micro/go-micro/v2/flow/service/proto"
	"github.com/micro/go-micro/v2/logger"
	rmemory "github.com/micro/go-micro/v2/registry/memory"
	"github.com/micro/go-micro/v2/server"
	"github.com/micro/go-micro/v2/store"
	smemory "github.com/micro/go-micro/v2/store/memory"
	tmemory "github.com/micro/go-micro/v2/transport/memory"
)

type handler struct {
	ctx context.Context
	brk broker.Broker
	cli client.Client
}

func (h *handler) AccountCreate(evt broker.Event) error {
	topic, sendRsp := evt.Message().Header["Micro-Callback"]

	if sendRsp {
		pub := micro.NewEvent(topic, h.cli)
		if err := pub.Publish(h.ctx, &proto.Test{Name: "AccountCreate"}); err != nil {
			return err
		}
	}

	return evt.Ack()
}

func (h *handler) ContactCreate(evt broker.Event) error {
	topic, sendRsp := evt.Message().Header["Micro-Callback"]

	if sendRsp {
		pub := micro.NewEvent(topic, h.cli)
		if err := pub.Publish(h.ctx, &proto.Test{Name: "ContactCreate"}); err != nil {
			return err
		}
	}

	return evt.Ack()
}

func TestClientCall(t *testing.T) {
	logger.DefaultLogger = logger.NewLogger(logger.WithLevel(logger.TraceLevel))
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fl := flow.NewFlow(
		flow.WithStateStore(smemory.NewStore(store.Namespace("state"))),
		flow.WithDataStore(smemory.NewStore(store.Namespace("data"))),
		flow.WithFlowStore(smemory.NewStore(store.Namespace("flow"))),
	)

	if err = fl.Init(); err != nil {
		t.Fatal(err)
	}
	if err = fl.CreateStep("test_flow", &flow.Step{
		ID:        "cms_account.AccountService.AccountCreate",
		Operation: flow.EmptyOperation("cms_account.AccountService.AccountCreate"),
		After:     nil,
		Before:    nil,
	}); err != nil {
		t.Fatal(err)
	}
	if err = fl.CreateStep("test_flow", &flow.Step{
		ID:        "cms_account.AccountService.AccountDelete",
		Operation: flow.EmptyOperation("cms_account.AccountService.AccountDelete"),
		After:     nil,
		Before:    nil,
	}); err != nil {
		t.Fatal(err)
	}
	if err = fl.CreateStep("reverse", &flow.Step{
		ID:        "cms_account.AccountService.AccountDelete",
		Operation: flow.EmptyOperation("cms_account.AccountService.AccountDelete"),
		After:     nil,
		Before:    nil,
	}); err != nil {
		t.Fatal(err)
	}
	if err = fl.CreateStep("test_flow", &flow.Step{
		ID:        "cms_contact.ContactService.ContactCreate",
		Operation: flow.EmptyOperation("cms_contact.ContactService.ContactCreate"),
		After:     []string{"cms_account.AccountService.AccountCreate"},
		Before:    nil,
		Fallback: flow.FlowExecuteOperation("reverse",
			flow.EmptyOperation("cms_account.AccountService.AccountDelete").Name(),
		),
	}); err != nil {
		t.Fatal(err)
	}
	if err = fl.CreateStep("test_flow", &flow.Step{
		ID:        "cms_project.ProjectService.ProjectCreate",
		Operation: flow.EmptyOperation("cms_project.ProjectService.ProjectCreate"),
		After:     []string{"cms_account.AccountService.AccountCreate"},
		Before:    nil,
	}); err != nil {
		t.Fatal(err)
	}
	if err = fl.CreateStep("test_flow", &flow.Step{
		ID:        "cms_network.NetworkService.NetworkCreate",
		Operation: flow.EmptyOperation("cms_network.NetworkService.NetworkCreate"),
		After:     []string{"cms_account.AccountService.AccountCreate"},
		Before:    nil,
	}); err != nil {
		t.Fatal(err)
	}
	if err = fl.CreateStep("test_flow", &flow.Step{
		ID:        "cms_authz.AuthzService.AuthzCreate",
		Operation: flow.EmptyOperation("cms_authz.AuthzService.AuthzCreate"),
		After:     []string{"cms_account.AccountService.AccountCreate"},
		Before:    nil,
	}); err != nil {
		t.Fatal(err)
	}
	if err = fl.CreateStep("test_flow", &flow.Step{
		ID:        "cms_mailer.MailService.MailSend",
		Operation: flow.EmptyOperation("cms_mailer.MailService.MailSend"),
		After:     []string{"all"}, //[]string{"cms_account.AccountService.AccountCreate"},
		Before:    nil,
	}); err != nil {
		t.Fatal(err)
	}

	req := &proto.Test{Name: "client call test req"}
	rsp := &proto.Test{}
	//	err  = fl.
	rid, err := fl.Execute("test_flow", req, rsp,
		flow.ExecuteContext(ctx),
		flow.ExecuteAsync(false),
		flow.ExecuteStep("cms_account.AccountService.AccountCreate"),
		flow.ExecuteClient(client.DefaultClient),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("rid %s rsp: %#+v\n", rid, rsp)

}

func TestClientPubsub(t *testing.T) {
	logger.DefaultLogger = logger.NewLogger(logger.WithLevel(logger.TraceLevel))
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fl := flow.NewFlow(
		flow.WithStateStore(smemory.NewStore(store.Namespace("state"))),
		flow.WithDataStore(smemory.NewStore(store.Namespace("data"))),
		flow.WithFlowStore(smemory.NewStore(store.Namespace("flow"))),
	)

	if err = fl.Init(); err != nil {
		t.Fatal(err)
	}
	if err = fl.CreateStep("test_flow", &flow.Step{
		ID:        "cms_account.AccountService.AccountCreate",
		Operation: flow.ClientPublishOperation("cms_account.AccountService.AccountCreate"),
		After:     nil,
		Before:    nil,
	}); err != nil {
		t.Fatal(err)
	}
	if err = fl.CreateStep("test_flow", &flow.Step{
		ID:        "cms_contact.ContactService.ContactCreate",
		Operation: flow.ClientPublishOperation("cms_contact.ContactService.ContactCreate"),
		After:     []string{"cms_account.AccountService.AccountCreate"},
		Before:    nil,
	}); err != nil {
		t.Fatal(err)
	}

	reg := rmemory.NewRegistry()
	brk := mbroker.NewBroker(broker.Registry(reg))
	brk.Connect()
	tr := tmemory.NewTransport()
	cli := client.NewClient(
		client.Selector(rselector.NewSelector(selector.Registry(reg))),
		client.Registry(reg), client.Transport(tr), client.Broker(brk))

	srv1 := server.NewServer(server.Name("cms_account"), server.Registry(reg), server.Transport(tr), server.Broker(brk))
	h1 := &handler{ctx: ctx, cli: cli, brk: brk}
	sub1, err := cli.Options().Broker.Subscribe("cms_account.AccountService.AccountCreate", h1.AccountCreate)
	if err != nil {
		t.Fatalf("failed to sub1: %v", err)
	}
	defer func() {
		if err := sub1.Unsubscribe(); err != nil {
			t.Fatal(err)
		}
	}()
	if err := srv1.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	srv2 := server.NewServer(server.Name("cms_contact"), server.Registry(reg), server.Transport(tr), server.Broker(brk))
	h2 := &handler{ctx: ctx, cli: cli, brk: brk}
	sub2, err := cli.Options().Broker.Subscribe("cms_contact.ContactService.ContactCreate", h2.ContactCreate)
	if err != nil {
		t.Fatalf("failed to sub1: %v", err)
	}
	defer func() {
		if err := sub2.Unsubscribe(); err != nil {
			t.Fatal(err)
		}
	}()
	if err := srv2.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	req := &proto.Test{Name: "req"}
	rsp := &proto.Test{}
	//	err  = fl.
	rid, err := fl.Execute("test_flow", req, rsp,
		flow.ExecuteContext(ctx),
		flow.ExecuteAsync(false),
		flow.ExecuteStep("cms_account.AccountService.AccountCreate"),
		flow.ExecuteClient(cli),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("rid %s rsp %#+v\n", rid, rsp)
	//	time.Sleep(5 * time.Second)
}

func BenchmarkFlowExecution(b *testing.B) {
	var err error
	ctx := context.Background()

	fl := flow.NewFlow(
		flow.WithStateStore(smemory.NewStore(store.Namespace("state"))),
		flow.WithDataStore(smemory.NewStore(store.Namespace("data"))),
		flow.WithFlowStore(smemory.NewStore(store.Namespace("flow"))),
	)

	if err = fl.Init(); err != nil {
		b.Fatal(err)
	}
	if err = fl.CreateStep("test_flow", &flow.Step{
		ID:        "cms_account.AccountService.AccountCreate",
		Operation: flow.EmptyOperation("cms_account.AccountService.AccountCreate"),
		After:     nil,
		Before:    nil,
	}); err != nil {
		b.Fatal(err)
	}
	if err = fl.CreateStep("test_flow", &flow.Step{
		ID:        "cms_account.AccountService.AccountDelete",
		Operation: flow.EmptyOperation("cms_account.AccountService.AccountDelete"),
		After:     nil,
		Before:    nil,
	}); err != nil {
		b.Fatal(err)
	}
	if err = fl.CreateStep("reverse", &flow.Step{
		ID:        "cms_account.AccountService.AccountDelete",
		Operation: flow.EmptyOperation("cms_account.AccountService.AccountDelete"),
		After:     nil,
		Before:    nil,
	}); err != nil {
		b.Fatal(err)
	}
	if err = fl.CreateStep("test_flow", &flow.Step{
		ID:        "cms_contact.ContactService.ContactCreate",
		Operation: flow.EmptyOperation("cms_contact.ContactService.ContactCreate"),
		After:     []string{"cms_account.AccountService.AccountCreate"},
		Before:    nil,
		Fallback: flow.FlowExecuteOperation("reverse",
			flow.EmptyOperation("cms_account.AccountService.AccountDelete").Name(),
		),
	}); err != nil {
		b.Fatal(err)
	}
	if err = fl.CreateStep("test_flow", &flow.Step{
		ID:        "cms_project.ProjectService.ProjectCreate",
		Operation: flow.EmptyOperation("cms_project.ProjectService.ProjectCreate"),
		After:     []string{"cms_account.AccountService.AccountCreate"},
		Before:    nil,
	}); err != nil {
		b.Fatal(err)
	}
	if err = fl.CreateStep("test_flow", &flow.Step{
		ID:        "cms_network.NetworkService.NetworkCreate",
		Operation: flow.EmptyOperation("cms_network.NetworkService.NetworkCreate"),
		After:     []string{"cms_account.AccountService.AccountCreate"},
		Before:    nil,
	}); err != nil {
		b.Fatal(err)
	}
	if err = fl.CreateStep("test_flow", &flow.Step{
		ID:        "cms_authz.AuthzService.AuthzCreate",
		Operation: flow.EmptyOperation("cms_authz.AuthzService.AuthzCreate"),
		After:     []string{"cms_account.AccountService.AccountCreate"},
		Before:    nil,
	}); err != nil {
		b.Fatal(err)
	}
	if err = fl.CreateStep("test_flow", &flow.Step{
		ID:        "cms_mailer.MailService.MailSend",
		Operation: flow.EmptyOperation("cms_mailer.MailService.MailSend"),
		After:     []string{"all"}, //[]string{"cms_account.AccountService.AccountCreate"},
		Before:    nil,
	}); err != nil {
		b.Fatal(err)
	}

	req := &proto.Test{Name: "req"}
	rsp := &proto.Test{}
	//	err  = fl.
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rid, err := fl.Execute("test_flow", req, rsp,
			flow.ExecuteContext(ctx),
			flow.ExecuteStep("cms_account.AccountService.AccountCreate"),
			flow.ExecuteAsync(false),
			flow.ExecuteClient(client.DefaultClient),
		)
		if err != nil {
			b.Fatal(err)
		}
		_ = rid
	}
}
