package a2a

import (
	"context"
	"testing"
)

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
