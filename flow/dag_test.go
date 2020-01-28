package flow_test

import (
	"context"
	"testing"
	"time"

	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/flow"
	proto "github.com/micro/go-micro/flow/service/proto"
	memory "github.com/micro/go-micro/flow/store/memory"
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
		ID: "cms_account.AccountService.AccountCreate",
		Operations: []flow.Operation{
			flow.ClientCallOperation("cms_account", "AccountService.AccountCreate"),
		},
		Requires: nil,
		Required: nil,
	}); err != nil {
		t.Fatal(err)
	}
	if err = fl.CreateStep("forward", &flow.Step{
		ID: "cms_account.AccountService.AccountDelete",
		Operations: []flow.Operation{
			flow.ClientCallOperation("cms_account", "AccountService.AccountDelete"),
		},
		Requires: nil,
		Required: nil,
	}); err != nil {
		t.Fatal(err)
	}
	if err = fl.CreateStep("reverse", &flow.Step{
		ID: "cms_account.AccountService.AccountDelete",
		Operations: []flow.Operation{
			flow.ClientCallOperation("cms_account", "AccountService.AccountDelete"),
		},
		Requires: nil,
		Required: nil,
	}); err != nil {
		t.Fatal(err)
	}
	if err = fl.CreateStep("forward", &flow.Step{
		ID: "cms_contact.ContactService.ContactCreate",
		Operations: []flow.Operation{
			flow.ClientCallOperation("cms_contact", "ContactService.ContactCreate"),
		},
		Requires: []string{"cms_account.AccountService.AccountCreate"},
		Required: nil,
		Fallback: []flow.Operation{
			flow.FlowExecuteOperation("reverse",
				flow.ClientCallOperation("cms_account", "AccountService.AccountDelete").Name(),
			),
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err = fl.CreateStep("forward", &flow.Step{
		ID: "cms_project.ProjectService.ProjectCreate",
		Operations: []flow.Operation{
			flow.ClientCallOperation("cms_project", "ProjectService.ProjectCreate"),
		},
		Requires: []string{"cms_account.AccountService.AccountCreate"},
		Required: nil,
	}); err != nil {
		t.Fatal(err)
	}
	if err = fl.CreateStep("forward", &flow.Step{
		ID: "cms_network.NetworkService.NetworkCreate",
		Operations: []flow.Operation{
			flow.ClientCallOperation("cms_network", "NetworkService.NetworkCreate"),
		},
		Requires: []string{"cms_account.AccountService.AccountCreate"},
		Required: nil,
	}); err != nil {
		t.Fatal(err)
	}
	if err = fl.CreateStep("forward", &flow.Step{
		ID: "cms_authz.AuthzService.AuthzCreate",
		Operations: []flow.Operation{
			flow.ClientCallOperation("cms_authz", "AuthzService.AuthzCreate"),
		},
		Requires: []string{"cms_account.AccountService.AccountCreate"},
		Required: nil,
	}); err != nil {
		t.Fatal(err)
	}
	if err = fl.CreateStep("forward", &flow.Step{
		ID: "cms_mailer.MailService.MailSend",
		Operations: []flow.Operation{
			flow.ClientCallOperation("cms_mailer", "MailService.MailSend"),
		},
		Requires: []string{"all"}, //[]string{"cms_account.AccountService.AccountCreate"},
		Required: nil,
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
	t.Logf("rid %s", rid)

	/*
		vs, err := getVertex(g, flow.ClientCallOperation("cms_mailer", "MailerService.MailSend").String())
		if err != nil {
			t.Fatal(err)
		}
		tr = &walk{}
		err = g.ReverseDepthFirstWalk([]dag.Vertex{vs}, tr.Walk)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("backward after\n")
		sort.Slice(tr.nodes, func(i, j int) bool {
			return tr.nodes[i].pos < tr.nodes[j].pos
		})
		for _, n := range tr.nodes {
			fmt.Printf("node: %v pos: %d\n", n.item, n.pos)
		}
	*/
}
