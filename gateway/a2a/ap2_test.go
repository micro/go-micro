package a2a

import (
	"crypto/ed25519"
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
