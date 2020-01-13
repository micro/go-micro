// +build ignore

package flow

import (
	"math/rand"
	"testing"
	"time"

	log "github.com/micro/go-micro/util/log"
)

func TestFlow(t *testing.T) {
	rand.Seed(time.Now().UnixNano())

	log.SetLevel(log.LevelError)
	//failHandler :=
	exc := NewExecutor(
		ExecutorStateStore(NewStateStore()),
		ExecutorDataStore(NewDataStore()),
		ExecutorFlowStore(NewFlowStore()),
		ExecutorLogger(NewLogger(log.LevelError)),
		ExecutorTimeout(5*time.Second),
		ExecutorConcurrency(100),
		//		ExecutorFailHandler(failHandler),
		//		ExecutorEventHandler(&microEvent{db: db}),
		//ExecutorStateChan(stateChan),
	)

	flowStore, err := exc.GetFlowStore()
	if err != nil {
		t.Fatal(err)
	}

	operations := []*SagaOperation{
		&SagaOperation{
			Forward: &FlowOperation{
				Node:     "unistack_cms_account",
				Service:  "AccountService",
				Endpoint: "AccountCreate",
			}},
		&SagaOperation{
			Forward: &FlowOperation{
				Node:     "unistack_cms_authz",
				Service:  "AuthzService",
				Endpoint: "AuthzCreate",
				//Recovery: "AuthzDelete",
				Requires: []string{"unistack_cms_account.AccountService.AccountCreate"},
			}},
		&SagaOperation{
			Forward: &FlowOperation{
				Node:     "unistack_cms_contact",
				Service:  "ContactService",
				Endpoint: "ContactCreate",
				//Recovery: "ContactDelete",
				Requires: []string{"unistack_cms_account.AccountService.AccountCreate"},
			}},
		&SagaOperation{
			Forward: &FlowOperation{
				Node:     "unistack_cms_network",
				Service:  "NetworkService",
				Endpoint: "NetworkCreate",
				Requires: []string{"unistack_cms_account.AccountService.AccountCreate"},
			}},
		&SagaOperation{
			Forward: &FlowOperation{
				Node:     "unistack_cms_project",
				Service:  "ProjectService",
				Endpoint: "ProjectCreate",
				Requires: []string{"unistack_cms_account.AccountService.AccountCreate"},
			},
			Reverse:  &FlowOperation{},
			Node:     "unistack_cms_project",
			Service:  "ProjectService",
			Endpoint: "ProjectDelete",
			Requires: []string{"unistack_cms_account.AccountService.AccountCreate"},
		},
		&SagaOperation{
			Forward: &FlowOperation{
				Node:      "unistack_cms_mailer",
				Service:   "MailerService",
				Endpoint:  "MailerSend",
				Aggregate: true,
			}},
	}

	err = flowStore.Save("forward", operations)
	if err != nil {
		t.Fatal(err)
	}

	if _, _, err = exc.Execute("forward", []byte("test req1")); err != nil {
		t.Fatal(err)
	}

}
