package a2a

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
