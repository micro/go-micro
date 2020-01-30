package flow_test

import (
	"context"
	"testing"
	"time"

	"github.com/micro/go-micro/v2/broker"
	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/flow"
	proto "github.com/micro/go-micro/v2/flow/service/proto"
	memory "github.com/micro/go-micro/v2/flow/store/memory"
)

func TestExecutor(t *testing.T) {
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sStore := memory.NewDataStore()
	dStore := memory.NewDataStore()
	fStore := memory.NewFlowStore()

	fl := flow.NewFlow(
		flow.WithStateStore(sStore),
		flow.WithDataStore(dStore),
		flow.WithFlowStore(fStore),
	)

	if err = fl.Init(); err != nil {
		t.Fatal(err)
	}
	if err = fl.CreateStep("forward", &flow.Step{
		ID:        "cms_account.AccountService.AccountCreate",
		Operation: flow.ClientCallOperation("cms_account", "AccountService.AccountCreate"),
		Requires:  nil,
		Required:  nil,
	}); err != nil {
		t.Fatal(err)
	}
	if err = fl.CreateStep("forward", &flow.Step{
		ID:        "cms_account.AccountService.AccountDelete",
		Operation: flow.ClientCallOperation("cms_account", "AccountService.AccountDelete"),
		Requires:  nil,
		Required:  nil,
	}); err != nil {
		t.Fatal(err)
	}
	if err = fl.CreateStep("reverse", &flow.Step{
		ID:        "cms_account.AccountService.AccountDelete",
		Operation: flow.ClientCallOperation("cms_account", "AccountService.AccountDelete"),
		Requires:  nil,
		Required:  nil,
	}); err != nil {
		t.Fatal(err)
	}
	if err = fl.CreateStep("forward", &flow.Step{
		ID:        "cms_contact.ContactService.ContactCreate",
		Operation: flow.ClientCallOperation("cms_contact", "ContactService.ContactCreate"),
		Requires:  []string{"cms_account.AccountService.AccountCreate"},
		Required:  nil,
		Fallback: flow.FlowExecuteOperation("reverse",
			flow.ClientCallOperation("cms_account", "AccountService.AccountDelete").Name(),
		),
	}); err != nil {
		t.Fatal(err)
	}
	if err = fl.CreateStep("forward", &flow.Step{
		ID:        "cms_project.ProjectService.ProjectCreate",
		Operation: flow.ClientCallOperation("cms_project", "ProjectService.ProjectCreate"),
		Requires:  []string{"cms_account.AccountService.AccountCreate"},
		Required:  nil,
	}); err != nil {
		t.Fatal(err)
	}
	if err = fl.CreateStep("forward", &flow.Step{
		ID:        "cms_network.NetworkService.NetworkCreate",
		Operation: flow.ClientCallOperation("cms_network", "NetworkService.NetworkCreate"),
		Requires:  []string{"cms_account.AccountService.AccountCreate"},
		Required:  nil,
	}); err != nil {
		t.Fatal(err)
	}
	if err = fl.CreateStep("forward", &flow.Step{
		ID:        "cms_authz.AuthzService.AuthzCreate",
		Operation: flow.ClientCallOperation("cms_authz", "AuthzService.AuthzCreate"),
		Requires:  []string{"cms_account.AccountService.AccountCreate"},
		Required:  nil,
	}); err != nil {
		t.Fatal(err)
	}
	if err = fl.CreateStep("forward", &flow.Step{
		ID:        "cms_mailer.MailService.MailSend",
		Operation: flow.ClientCallOperation("cms_mailer", "MailService.MailSend"),
		Requires:  []string{"all"}, //[]string{"cms_account.AccountService.AccountCreate"},
		Required:  nil,
	}); err != nil {
		t.Fatal(err)
	}

	req := &proto.Test{Name: "req"}
	rsp := &proto.Test{}
	//	err  = fl.
	rid, err := fl.Execute("forward", "cms_account.AccountService.AccountCreate", req, rsp,
		flow.ExecuteContext(ctx),
		flow.ExecuteAsync(false),
		flow.ExecuteClient(client.DefaultClient),
		flow.ExecuteBroker(broker.DefaultBroker),
	)
	if err != nil {
		t.Fatal(err)
	}
	_ = rid
	//	t.Logf("rid %s", rid)
}

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
		Requires:  nil,
		Required:  nil,
	}); err != nil {
		b.Fatal(err)
	}
	if err = fl.CreateStep("forward", &flow.Step{
		ID:        "cms_account.AccountService.AccountDelete",
		Operation: flow.ClientCallOperation("cms_account", "AccountService.AccountDelete"),
		Requires:  nil,
		Required:  nil,
	}); err != nil {
		b.Fatal(err)
	}
	if err = fl.CreateStep("reverse", &flow.Step{
		ID:        "cms_account.AccountService.AccountDelete",
		Operation: flow.ClientCallOperation("cms_account", "AccountService.AccountDelete"),
		Requires:  nil,
		Required:  nil,
	}); err != nil {
		b.Fatal(err)
	}
	if err = fl.CreateStep("forward", &flow.Step{
		ID:        "cms_contact.ContactService.ContactCreate",
		Operation: flow.ClientCallOperation("cms_contact", "ContactService.ContactCreate"),
		Requires:  []string{"cms_account.AccountService.AccountCreate"},
		Required:  nil,
		Fallback: flow.FlowExecuteOperation("reverse",
			flow.ClientCallOperation("cms_account", "AccountService.AccountDelete").Name(),
		),
	}); err != nil {
		b.Fatal(err)
	}
	if err = fl.CreateStep("forward", &flow.Step{
		ID:        "cms_project.ProjectService.ProjectCreate",
		Operation: flow.ClientCallOperation("cms_project", "ProjectService.ProjectCreate"),
		Requires:  []string{"cms_account.AccountService.AccountCreate"},
		Required:  nil,
	}); err != nil {
		b.Fatal(err)
	}
	if err = fl.CreateStep("forward", &flow.Step{
		ID:        "cms_network.NetworkService.NetworkCreate",
		Operation: flow.ClientCallOperation("cms_network", "NetworkService.NetworkCreate"),
		Requires:  []string{"cms_account.AccountService.AccountCreate"},
		Required:  nil,
	}); err != nil {
		b.Fatal(err)
	}
	if err = fl.CreateStep("forward", &flow.Step{
		ID:        "cms_authz.AuthzService.AuthzCreate",
		Operation: flow.ClientCallOperation("cms_authz", "AuthzService.AuthzCreate"),
		Requires:  []string{"cms_account.AccountService.AccountCreate"},
		Required:  nil,
	}); err != nil {
		b.Fatal(err)
	}
	if err = fl.CreateStep("forward", &flow.Step{
		ID:        "cms_mailer.MailService.MailSend",
		Operation: flow.ClientCallOperation("cms_mailer", "MailService.MailSend"),
		Requires:  []string{"all"}, //[]string{"cms_account.AccountService.AccountCreate"},
		Required:  nil,
	}); err != nil {
		b.Fatal(err)
	}

	req := &proto.Test{Name: "req"}
	rsp := &proto.Test{}
	//	err  = fl.
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rid, err := fl.Execute("forward", "cms_account.AccountService.AccountCreate", req, rsp,
			flow.ExecuteContext(ctx),
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
