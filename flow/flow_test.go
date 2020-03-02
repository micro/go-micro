package flow_test

import (
	"context"
	"testing"
	"time"

	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/flow"
	proto "github.com/micro/go-micro/v2/flow/service/proto"
	"github.com/micro/go-micro/v2/store"
	smemory "github.com/micro/go-micro/v2/store/memory"
)

func TestExecutor(t *testing.T) {
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
	if err = fl.CreateStep("forward", &flow.Step{
		ID:        "cms_account.AccountService.AccountCreate",
		Operation: flow.EmptyOperation("cms_account.AccountService.AccountCreate"),
		After:     nil,
		Before:    nil,
	}); err != nil {
		t.Fatal(err)
	}
	if err = fl.CreateStep("forward", &flow.Step{
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
	if err = fl.CreateStep("forward", &flow.Step{
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
	if err = fl.CreateStep("forward", &flow.Step{
		ID:        "cms_project.ProjectService.ProjectCreate",
		Operation: flow.EmptyOperation("cms_project.ProjectService.ProjectCreate"),
		After:     []string{"cms_account.AccountService.AccountCreate"},
		Before:    nil,
	}); err != nil {
		t.Fatal(err)
	}
	if err = fl.CreateStep("forward", &flow.Step{
		ID:        "cms_network.NetworkService.NetworkCreate",
		Operation: flow.EmptyOperation("cms_network.NetworkService.NetworkCreate"),
		After:     []string{"cms_account.AccountService.AccountCreate"},
		Before:    nil,
	}); err != nil {
		t.Fatal(err)
	}
	if err = fl.CreateStep("forward", &flow.Step{
		ID:        "cms_authz.AuthzService.AuthzCreate",
		Operation: flow.EmptyOperation("cms_authz.AuthzService.AuthzCreate"),
		After:     []string{"cms_account.AccountService.AccountCreate"},
		Before:    nil,
	}); err != nil {
		t.Fatal(err)
	}
	if err = fl.CreateStep("forward", &flow.Step{
		ID:        "cms_mailer.MailService.MailSend",
		Operation: flow.EmptyOperation("cms_mailer.MailService.MailSend"),
		After:     []string{"all"}, //[]string{"cms_account.AccountService.AccountCreate"},
		Before:    nil,
	}); err != nil {
		t.Fatal(err)
	}

	req := &proto.Test{Name: "req"}
	rsp := &proto.Test{}
	//	err  = fl.
	rid, err := fl.Execute("forward", req, rsp,
		flow.ExecuteContext(ctx),
		flow.ExecuteAsync(false),
		flow.ExecuteStep("cms_account.AccountService.AccountCreate"),
		flow.ExecuteClient(client.DefaultClient),
	)
	if err != nil {
		t.Fatal(err)
	}
	_ = rid
	//	t.Logf("rid %s", rid)
}

/*
func BenchmarkFlowExecution(b *testing.B) {
	var err error
	ctx := context.Background()

	sStore := memory.NewDataStore()
	dStore := memory.NewDataStore()
	fStore := memory.NewFlowStore()

	fl := flow.NewFlow(
		flow.WithStateStore(sStore),
		flow.WithDataStore(dStore),
		flow.WithFlowStore(fStore),
	)

	if err = fl.Init(); err != nil {
		b.Fatal(err)
	}
	if err = fl.CreateStep("forward", &flow.Step{
		ID:        "cms_account.AccountService.AccountCreate",
		Operation: flow.ClientCallOperation("cms_account", "AccountService.AccountCreate"),
		After:     nil,
		Before:    nil,
	}); err != nil {
		b.Fatal(err)
	}
	if err = fl.CreateStep("forward", &flow.Step{
		ID:        "cms_account.AccountService.AccountDelete",
		Operation: flow.ClientCallOperation("cms_account", "AccountService.AccountDelete"),
		After:     nil,
		Before:    nil,
	}); err != nil {
		b.Fatal(err)
	}
	if err = fl.CreateStep("reverse", &flow.Step{
		ID:        "cms_account.AccountService.AccountDelete",
		Operation: flow.ClientCallOperation("cms_account", "AccountService.AccountDelete"),
		After:     nil,
		Before:    nil,
	}); err != nil {
		b.Fatal(err)
	}
	if err = fl.CreateStep("forward", &flow.Step{
		ID:        "cms_contact.ContactService.ContactCreate",
		Operation: flow.ClientCallOperation("cms_contact", "ContactService.ContactCreate"),
		After:     []string{"cms_account.AccountService.AccountCreate"},
		Before:    nil,
		Fallback: flow.FlowExecuteOperation("reverse",
			flow.ClientCallOperation("cms_account", "AccountService.AccountDelete").Name(),
		),
	}); err != nil {
		b.Fatal(err)
	}
	if err = fl.CreateStep("forward", &flow.Step{
		ID:        "cms_project.ProjectService.ProjectCreate",
		Operation: flow.ClientCallOperation("cms_project", "ProjectService.ProjectCreate"),
		After:     []string{"cms_account.AccountService.AccountCreate"},
		Before:    nil,
	}); err != nil {
		b.Fatal(err)
	}
	if err = fl.CreateStep("forward", &flow.Step{
		ID:        "cms_network.NetworkService.NetworkCreate",
		Operation: flow.ClientCallOperation("cms_network", "NetworkService.NetworkCreate"),
		After:     []string{"cms_account.AccountService.AccountCreate"},
		Before:    nil,
	}); err != nil {
		b.Fatal(err)
	}
	if err = fl.CreateStep("forward", &flow.Step{
		ID:        "cms_authz.AuthzService.AuthzCreate",
		Operation: flow.ClientCallOperation("cms_authz", "AuthzService.AuthzCreate"),
		After:     []string{"cms_account.AccountService.AccountCreate"},
		Before:    nil,
	}); err != nil {
		b.Fatal(err)
	}
	if err = fl.CreateStep("forward", &flow.Step{
		ID:        "cms_mailer.MailService.MailSend",
		Operation: flow.ClientCallOperation("cms_mailer", "MailService.MailSend"),
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
		rid, err := fl.Execute("forward", req, rsp,
			flow.ExecuteContext(ctx),
			flow.ExecuteStep("cms_account.AccountService.AccountCreate"),
			flow.ExecuteAsync(false),
			flow.ExecuteClient(client.DefaultClient),
			flow.ExecuteBroker(broker.DefaultBroker),
		)
		if err != nil {
			b.Fatal(err)
		}
		_ = rid
	}
}
*/
