// +build ignore

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

func TestReverse(t *testing.T) {
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
		ID:        "cms_instance_backend.BackendService.InstanceDelete",
		Operation: flow.EmptyOperation("cms_instance_backend.BackendService.InstanceDelete"),
		After:     nil,
		Before:    []string{"cms_instance.InstanceService.InstanceDelete"},
	}); err != nil {
		t.Fatal(err)
	}
	if err = fl.CreateStep("test_flow", &flow.Step{
		ID:        "cms_instance.InstanceService.InstanceDelete",
		Operation: flow.EmptyOperation("cms_instance.InstanceService.InstanceDelete"),
		After:     []string{"cms_instance_backend.BackendService.InstanceDelete"},
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
		flow.ExecuteStep("cms_instance.InstanceService.InstanceDelete"),
		flow.ExecuteClient(client.DefaultClient),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("rid %s rsp: %#+v\n", rid, rsp)

}
