// +build ignore

package flow_test

import (
	"context"
	"fmt"
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

	steps []string
}

func (h *handler) AccountCreate(evt broker.Event) error {
	topic, sendRsp := evt.Message().Header["Micro-Callback"]
	if sendRsp {
		pub := micro.NewEvent(topic, h.cli)
		if err := pub.Publish(h.ctx, &proto.Test{Name: "AccountCreate"}); err != nil {
			return err
		}
	}
	h.steps = append(h.steps, "AccountCreate")
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
	h.steps = append(h.steps, "ContactCreate")
	return evt.Ack()
}

func (h *handler) NotifyEmail(evt broker.Event) error {
	topic, sendRsp := evt.Message().Header["Micro-Callback"]
	if sendRsp {
		pub := micro.NewEvent(topic, h.cli)
		if err := pub.Publish(h.ctx, &proto.Test{Name: "NotifyEmail"}); err != nil {
			return err
		}
	}
	h.steps = append(h.steps, "NotifyEmail")
	return evt.Ack()
}
func (h *handler) Failure(evt broker.Event) error {
	topic, sendRsp := evt.Message().Header["Micro-Callback"]
	if sendRsp {
		pub := micro.NewEvent(topic, h.cli)
		if err := pub.Publish(h.ctx, &proto.Test{Name: "Failure"}); err != nil {
			return err
		}
	}
	h.steps = append(h.steps, "Failure")
	evt.Ack()
	return fmt.Errorf("failure")
}
func (h *handler) Fallback(evt broker.Event) error {
	topic, sendRsp := evt.Message().Header["Micro-Callback"]
	if sendRsp {
		pub := micro.NewEvent(topic, h.cli)
		if err := pub.Publish(h.ctx, &proto.Test{Name: "Fallback"}); err != nil {
			return err
		}
	}
	h.steps = append(h.steps, "Fallback")
	return evt.Ack()
}

func TestClientPubsub(t *testing.T) {
	logger.DefaultLogger = logger.NewLogger(logger.WithLevel(logger.InfoLevel))
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
		ID:        "AccountCreate",
		Operation: flow.ClientPublishOperation("AccountCreate"),
		After:     nil,
		Before:    nil,
	}); err != nil {
		t.Fatal(err)
	}
	if err = fl.CreateStep("test_flow", &flow.Step{
		ID:        "ContactCreate",
		Operation: flow.ClientPublishOperation("ContactCreate"),
		After:     []string{"AccountCreate"},
		Before:    nil,
	}); err != nil {
		t.Fatal(err)
	}
	if err = fl.CreateStep("test_flow", &flow.Step{
		ID:        "NotifyEmail",
		Operation: flow.ClientPublishOperation("NotifyEmail"),
		After:     []string{"ContactCreate"},
		Before:    nil,
	}); err != nil {
		t.Fatal(err)
	}
	if err = fl.CreateStep("test_flow", &flow.Step{
		ID:        "Failure",
		Operation: flow.ClientPublishOperation("Failure"),
		After:     []string{"NotifyEmail"},
		Before:    nil,
		Fallback:  flow.FlowExecuteOperation("test_flow", "Fallback"),
	}); err != nil {
		t.Fatal(err)
	}
	if err = fl.CreateStep("test_flow", &flow.Step{
		ID:        "Fallback",
		Operation: flow.ClientPublishOperation("Fallback"),
		After:     nil,
		Before:    nil,
	}); err != nil {
		t.Fatal(err)
	}

	reg := rmemory.NewRegistry()
	brk := mbroker.NewBroker(broker.Registry(reg))
	brk.Connect()
	tr := tmemory.NewTransport()
	cli := client.NewClient(
		client.Retries(0),
		client.Selector(rselector.NewSelector(selector.Registry(reg))),
		client.Registry(reg), client.Transport(tr), client.Broker(brk))

	h := &handler{ctx: ctx, cli: cli, brk: brk}

	srv1 := server.NewServer(server.Name("account"), server.Registry(reg), server.Transport(tr), server.Broker(brk))
	sub1, err := cli.Options().Broker.Subscribe("AccountCreate", h.AccountCreate)
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

	srv2 := server.NewServer(server.Name("contact"), server.Registry(reg), server.Transport(tr), server.Broker(brk))
	sub2, err := cli.Options().Broker.Subscribe("ContactCreate", h.ContactCreate)
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
	srv3 := server.NewServer(server.Name("notify"), server.Registry(reg), server.Transport(tr), server.Broker(brk))
	sub3, err := cli.Options().Broker.Subscribe("NotifyEmail", h.NotifyEmail)
	if err != nil {
		t.Fatalf("failed to sub1: %v", err)
	}
	defer func() {
		if err := sub3.Unsubscribe(); err != nil {
			t.Fatal(err)
		}
	}()
	if err := srv3.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	srv4 := server.NewServer(server.Name("failure"), server.Registry(reg), server.Transport(tr), server.Broker(brk))
	sub4, err := cli.Options().Broker.Subscribe("Failure", h.Failure)
	if err != nil {
		t.Fatalf("failed to sub1: %v", err)
	}
	defer func() {
		if err := sub4.Unsubscribe(); err != nil {
			t.Fatal(err)
		}
	}()
	if err := srv4.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	srv5 := server.NewServer(server.Name("fallback"), server.Registry(reg), server.Transport(tr), server.Broker(brk))
	sub5, err := cli.Options().Broker.Subscribe("Fallback", h.Fallback)
	if err != nil {
		t.Fatalf("failed to sub1: %v", err)
	}
	defer func() {
		if err := sub5.Unsubscribe(); err != nil {
			t.Fatal(err)
		}
	}()
	if err := srv5.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	req := &proto.Test{Name: "req"}
	rsp := &proto.Test{}
	//	err  = fl.
	rid, err := fl.Execute("test_flow", req, rsp,
		flow.ExecuteContext(ctx),
		flow.ExecuteAsync(false),
		flow.ExecuteStep("AccountCreate"),
		flow.ExecuteClient(cli),
	)
	for _, s := range h.steps {
		t.Logf("steps in order %s\n", s)
	}

	if err != nil {
		t.Fatal(err)
	}

	t.Logf("rid %s rsp %#+v\n", rid, rsp)
	//	time.Sleep(5 * time.Second)
}

/*
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
*/
