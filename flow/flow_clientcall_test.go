package flow_test

import (
	"context"
	"testing"
	"time"

	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/flow"
	proto "github.com/micro/go-micro/v2/flow/service/proto"
	"github.com/micro/go-micro/v2/logger"
	"github.com/micro/go-micro/v2/store"
	smemory "github.com/micro/go-micro/v2/store/memory"
)

func TestClientCall(t *testing.T) {
	logger.DefaultLogger = logger.NewLogger(logger.WithLevel(logger.TraceLevel))
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err = flow.DefaultFlow.Init(
		flow.WithStore(smemory.NewStore(store.Namespace("flow"))),
	); err != nil {
		t.Fatal(err)
	}

	fl := flow.DefaultFlow

	if err = flow.DefaultExecutor.Init(
		flow.WithFlow(fl),
		flow.WithStateStore(smemory.NewStore(store.Namespace("state"))),
		flow.WithDataStore(smemory.NewStore(store.Namespace("data"))),
	); err != nil {
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
	rid, err := fl.Execute(req, rsp,
		flow.ExecuteContext(ctx),
		flow.ExecuteAsync(false),
		flow.ExecuteStep("cms_account.AccountService.AccountCreate"),
		flow.ExecuteClient(client.DefaultClient),
		flow.ExecuteFlow("test_flow"),
	)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("rid %s rsp: %#+v\n", rid, rsp)
}
