package a2a

import (
	"context"
	"net/http/httptest"
	"testing"
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
