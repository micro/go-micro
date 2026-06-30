package a2a

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// An agent that embeds NewAgentHandler is directly A2A-queryable — no
// gateway, no registry — and the client talks to it the same way.
func TestEmbeddedAgentHandler(t *testing.T) {
	card := Card("solo", "http://localhost:4000", "", []string{"task"})
	h := NewAgentHandler(card, func(_ context.Context, text string) (string, error) {
		return "echo:" + text, nil
	})
	ts := httptest.NewServer(h)
	defer ts.Close()

	cl := NewClient(ts.URL)
	got, err := cl.Card(context.Background())
	if err != nil || got.Name != "solo" {
		t.Fatalf("Card() = %+v, err %v", got, err)
	}
	reply, err := cl.Send(context.Background(), "hi")
	if err != nil || reply != "echo:hi" {
		t.Fatalf("Send = %q, err %v", reply, err)
	}
}

// The client fetches a card and sends a message to an agent served by the
// gateway — A2A end to end, both directions, in one process.
func TestClientSendAndCard(t *testing.T) {
	ts, cleanup := newGatewayWithAgent(t)
	defer cleanup()

	cl := NewClient(ts.URL + "/agents/echo")

	card, err := cl.Card(context.Background())
	if err != nil || card.Name != "echo" {
		t.Fatalf("Card() = %+v, err %v", card, err)
	}

	reply, err := cl.Send(context.Background(), "ping")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if reply != "pong" {
		t.Errorf("Send reply = %q, want pong", reply)
	}
}

func TestClientContinuesTaskAndConfiguresPush(t *testing.T) {
	card := Card("solo", "http://localhost:4000", "", []string{"task"})
	h := NewAgentHandler(card, func(_ context.Context, text string) (string, error) {
		return "echo:" + text, nil
	})
	ts := httptest.NewServer(h)
	defer ts.Close()

	updates := make(chan Task, 1)
	push := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var task Task
		if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
			t.Errorf("decode push task: %v", err)
			return
		}
		updates <- task
		w.WriteHeader(http.StatusAccepted)
	}))
	defer push.Close()

	cl := NewClient(ts.URL)
	first, err := cl.SendMessage(context.Background(), Message{
		Parts: []Part{{Kind: "text", Text: "one"}},
	})
	if err != nil {
		t.Fatalf("first SendMessage: %v", err)
	}
	second, err := cl.SendMessage(context.Background(), Message{
		TaskID:    first.ID,
		ContextID: first.ContextID,
		Parts:     []Part{{Kind: "text", Text: "two"}},
	})
	if err != nil {
		t.Fatalf("second SendMessage: %v", err)
	}
	if second.ID != first.ID || second.ContextID != first.ContextID || len(second.History) != 4 {
		t.Fatalf("continued task = %+v, first %+v", second, first)
	}
	cfg := PushNotificationConfig{URL: push.URL}
	if err := cl.SetPushNotificationConfig(context.Background(), second.ID, cfg); err != nil {
		t.Fatalf("SetPushNotificationConfig: %v", err)
	}
	got, err := cl.PushNotificationConfig(context.Background(), second.ID)
	if err != nil {
		t.Fatalf("PushNotificationConfig: %v", err)
	}
	if got.URL != push.URL {
		t.Fatalf("PushNotificationConfig URL = %q, want %q", got.URL, push.URL)
	}
	select {
	case update := <-updates:
		if update.ID != second.ID {
			t.Fatalf("push update ID = %q, want %q", update.ID, second.ID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for push update")
	}
}

func TestClientResubscribeStreamsRetainedAndLiveTask(t *testing.T) {
	d := newDispatcher()
	initial := &Task{ID: "task-1", ContextID: "ctx-1", Kind: "task", Status: TaskStatus{State: stateWorking, Timestamp: time.Now().UTC().Format(time.RFC3339)}}
	d.store(initial)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		d.serve(w, r, func(context.Context, string) (string, error) { return "", nil })
	}))
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	tasks, errs := NewClient(ts.URL).Resubscribe(ctx, initial.ID)

	first := <-tasks
	if first.ID != initial.ID || first.Status.State != stateWorking {
		t.Fatalf("first resubscribe task = %+v, want retained working task", first)
	}
	final := &Task{ID: initial.ID, ContextID: initial.ContextID, Kind: "task", Status: TaskStatus{State: stateCompleted, Timestamp: time.Now().UTC().Format(time.RFC3339)}, Artifacts: []Artifact{textArtifact("done")}}
	d.store(final)
	second := <-tasks
	if second.ID != final.ID || second.Status.State != stateCompleted || textOf(second.Artifacts[0].Parts) != "done" {
		t.Fatalf("second resubscribe task = %+v, want live completed task", second)
	}
	if _, ok := <-tasks; ok {
		t.Fatal("resubscribe task channel stayed open after terminal update")
	}
	select {
	case err := <-errs:
		if err != nil {
			t.Fatalf("resubscribe error = %v", err)
		}
	default:
	}
}

func TestClientSendMessageReturnsInputRequiredTask(t *testing.T) {
	card := Card("solo", "http://localhost:4000", "", []string{"task"})
	h := NewAgentHandler(card, func(context.Context, string) (string, error) {
		return "", errors.New("input-required: provide approval code")
	})
	ts := httptest.NewServer(h)
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	task, err := NewClient(ts.URL).SendMessage(ctx, Message{Parts: []Part{{Kind: "text", Text: "approve?"}}})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if task.Status.State != stateInputRequired || !strings.Contains(textOf(task.Artifacts[0].Parts), "provide approval code") {
		t.Fatalf("task = %+v, want input-required handoff", task)
	}
}
