package a2a

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func testAP2Key(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := NewAP2Keypair()
	if err != nil {
		t.Fatal(err)
	}
	return pub, priv
}

func TestAP2CheckoutMandateSignAttachAndVerify(t *testing.T) {
	pub, priv := testAP2Key(t)
	msg := Message{Role: "user", Kind: "message", TaskID: "task-1", ContextID: "ctx-1", Parts: []Part{{Kind: "text", Text: "buy"}}}
	mandate := AP2BindMandateToMessage(AP2Mandate{ID: "checkout-1", Kind: AP2CheckoutMandate, Subject: "alice", Merchant: "store", Amount: "10.00", Currency: "USD", Description: "demo", IssuedAt: time.Unix(1, 0).UTC()}, msg)
	signed, err := SignAP2Mandate(mandate, "test-key", priv)
	if err != nil {
		t.Fatal(err)
	}
	msg = AP2AttachMandate(msg, signed)
	task := taskFromReplyWithIDs(msg, "ok", stateCompleted, msg.TaskID, msg.ContextID)

	if len(task.AP2Mandates) != 1 {
		t.Fatalf("expected mandate carried on task, got %d", len(task.AP2Mandates))
	}
	got := VerifyAP2ForTask(task.AP2Mandates[0], pub, *task, nil)
	if !got.Verified || got.Error != "" {
		t.Fatalf("expected verified mandate, got %+v", got)
	}
}

func TestAP2PaymentMandateX402RailReference(t *testing.T) {
	pub, priv := testAP2Key(t)
	rail := X402AP2Rail("payreq_123")
	task := Task{ID: "task-2", ContextID: "ctx-2"}
	signed, err := SignAP2Mandate(AP2Mandate{ID: "payment-1", Kind: AP2PaymentMandate, TaskID: task.ID, ContextID: task.ContextID, Rail: &rail, IssuedAt: time.Unix(1, 0).UTC()}, "test-key", priv)
	if err != nil {
		t.Fatal(err)
	}
	got := VerifyAP2ForTask(signed, pub, task, &rail)
	if !got.Verified {
		t.Fatalf("expected x402 rail to verify, got %+v", got)
	}
}

// TestAP2GatewayVerifiesInboundPaymentMandate drives a real A2A message/send
// carrying a signed x402 payment mandate through the gateway and asserts the
// mandate is verified (and the x402 rail carried) into the task a paid path
// consults — and that a tampered mandate is surfaced as unverified.
func TestAP2GatewayVerifiesInboundPaymentMandate(t *testing.T) {
	pub, priv := testAP2Key(t)
	d := newDispatcher()
	d.ap2Verify = func(s AP2SignedMandate, task Task) AP2Verification {
		return VerifyAP2ForTask(s, pub, task, nil)
	}
	invoke := func(context.Context, string) (string, error) { return "fetched", nil }

	send := func(t *testing.T, mandate AP2SignedMandate) Task {
		t.Helper()
		msg := AP2AttachMandate(
			Message{Role: "user", Kind: "message", MessageID: "m1", Parts: []Part{{Kind: "text", Text: "pay and fetch"}}},
			mandate,
		)
		params, err := json.Marshal(sendParams{Message: msg})
		if err != nil {
			t.Fatal(err)
		}
		body := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"message/send","params":%s}`, params)
		return rpcTaskFromBody(t, d, body, invoke)
	}

	rail := X402AP2Rail("payreq_777")
	good, err := SignAP2Mandate(AP2Mandate{ID: "pay-1", Kind: AP2PaymentMandate, Rail: &rail, IssuedAt: time.Unix(1, 0).UTC()}, "k", priv)
	if err != nil {
		t.Fatal(err)
	}

	task := send(t, good)
	if len(task.AP2Verifications) != 1 || !task.AP2Verifications[0].Verified {
		t.Fatalf("inbound payment mandate not verified: %+v", task.AP2Verifications)
	}
	if task.AP2Verifications[0].Kind != string(AP2PaymentMandate) {
		t.Errorf("verification kind = %q, want payment", task.AP2Verifications[0].Kind)
	}
	if len(task.AP2Mandates) != 1 || task.AP2Mandates[0].Mandate.Rail == nil ||
		task.AP2Mandates[0].Mandate.Rail.Type != "x402" || task.AP2Mandates[0].Mandate.Rail.Reference != "payreq_777" {
		t.Fatalf("x402 settlement rail not carried onto task: %+v", task.AP2Mandates)
	}

	tampered := good
	tampered.Mandate.Amount = "999.00"
	bad := send(t, tampered)
	if len(bad.AP2Verifications) != 1 || bad.AP2Verifications[0].Verified {
		t.Fatalf("tampered mandate should be unverified: %+v", bad.AP2Verifications)
	}
	if !strings.Contains(bad.AP2Verifications[0].Error, "signature") {
		t.Errorf("tampered verification error = %q, want signature failure", bad.AP2Verifications[0].Error)
	}
}

// TestAP2CarriedUnverifiedWithoutKey confirms the default (no configured key)
// is unchanged: mandates are carried but not verified.
func TestAP2CarriedUnverifiedWithoutKey(t *testing.T) {
	_, priv := testAP2Key(t)
	d := newDispatcher() // no ap2Verify configured
	rail := X402AP2Rail("payreq_1")
	signed, err := SignAP2Mandate(AP2Mandate{ID: "pay-1", Kind: AP2PaymentMandate, Rail: &rail, IssuedAt: time.Unix(1, 0).UTC()}, "k", priv)
	if err != nil {
		t.Fatal(err)
	}
	msg := AP2AttachMandate(Message{Role: "user", Kind: "message", MessageID: "m1", Parts: []Part{{Kind: "text", Text: "x"}}}, signed)
	params, _ := json.Marshal(sendParams{Message: msg})
	body := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"message/send","params":%s}`, params)
	task := rpcTaskFromBody(t, d, body, func(context.Context, string) (string, error) { return "ok", nil })
	if len(task.AP2Mandates) != 1 {
		t.Fatalf("mandate should still be carried: %+v", task.AP2Mandates)
	}
	if len(task.AP2Verifications) != 0 {
		t.Errorf("no verifications without a configured key, got %+v", task.AP2Verifications)
	}
}

func TestAP2TamperCasesFailDistinctly(t *testing.T) {
	pub, priv := testAP2Key(t)
	rail := X402AP2Rail("payreq_123")
	task := Task{ID: "task-3", ContextID: "ctx-3"}
	signed, err := SignAP2Mandate(AP2Mandate{ID: "payment-2", Kind: AP2PaymentMandate, TaskID: task.ID, ContextID: task.ContextID, Rail: &rail, IssuedAt: time.Unix(1, 0).UTC()}, "test-key", priv)
	if err != nil {
		t.Fatal(err)
	}

	tampered := signed
	tampered.Mandate.Amount = "999.00"
	if got := VerifyAP2ForTask(tampered, pub, task, &rail); got.Verified || !strings.Contains(got.Error, "signature") {
		t.Fatalf("expected signature failure, got %+v", got)
	}

	wrongTask := task
	wrongTask.ID = "other-task"
	if got := VerifyAP2ForTask(signed, pub, wrongTask, &rail); got.Verified || !strings.Contains(got.Error, "task binding") {
		t.Fatalf("expected task binding failure, got %+v", got)
	}

	otherRail := X402AP2Rail("payreq_other")
	if got := VerifyAP2ForTask(signed, pub, task, &otherRail); got.Verified || !strings.Contains(got.Error, "rail reference") {
		t.Fatalf("expected rail reference failure, got %+v", got)
	}
}
