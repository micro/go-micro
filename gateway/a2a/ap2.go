package a2a

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// AP2MandateKind identifies the AP2 mandate stage represented by a credential.
type AP2MandateKind string

const (
	AP2CheckoutMandate AP2MandateKind = "checkout"
	AP2PaymentMandate  AP2MandateKind = "payment"
)

// AP2RailRef names the settlement rail authorized by an AP2 payment mandate.
type AP2RailRef struct {
	Type      string `json:"type"`
	Reference string `json:"reference"`
}

// AP2Mandate is a small verifiable AP2 credential. It is deliberately rail-neutral:
// x402 is represented as one possible rail reference under a payment mandate.
type AP2Mandate struct {
	ID          string         `json:"id"`
	Kind        AP2MandateKind `json:"kind"`
	Subject     string         `json:"subject,omitempty"`
	Merchant    string         `json:"merchant,omitempty"`
	Amount      string         `json:"amount,omitempty"`
	Currency    string         `json:"currency,omitempty"`
	Description string         `json:"description,omitempty"`
	TaskID      string         `json:"taskId,omitempty"`
	ContextID   string         `json:"contextId,omitempty"`
	Rail        *AP2RailRef    `json:"rail,omitempty"`
	IssuedAt    time.Time      `json:"issuedAt"`
}

// AP2SignedMandate is an AP2 mandate plus an Ed25519 signature over its canonical JSON.
type AP2SignedMandate struct {
	Mandate   AP2Mandate `json:"mandate"`
	KeyID     string     `json:"keyId,omitempty"`
	Signature string     `json:"signature"`
}

// AP2Verification records mandate verification on an A2A task without mixing in
// payment-settlement state.
type AP2Verification struct {
	MandateID string `json:"mandateId"`
	Kind      string `json:"kind"`
	Verified  bool   `json:"verified"`
	Error     string `json:"error,omitempty"`
}

// NewAP2Keypair returns an Ed25519 keypair suitable for tests or local demos.
func NewAP2Keypair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return ed25519.GenerateKey(rand.Reader)
}

// SignAP2Mandate signs a mandate as a verifiable AP2 credential.
func SignAP2Mandate(m AP2Mandate, keyID string, private ed25519.PrivateKey) (AP2SignedMandate, error) {
	if m.ID == "" {
		return AP2SignedMandate{}, errors.New("ap2: mandate id is required")
	}
	if m.Kind == "" {
		return AP2SignedMandate{}, errors.New("ap2: mandate kind is required")
	}
	if m.IssuedAt.IsZero() {
		m.IssuedAt = time.Now().UTC()
	}
	payload, err := ap2Payload(m)
	if err != nil {
		return AP2SignedMandate{}, err
	}
	return AP2SignedMandate{Mandate: m, KeyID: keyID, Signature: base64.RawURLEncoding.EncodeToString(ed25519.Sign(private, payload))}, nil
}

// VerifyAP2Mandate verifies a signed mandate credential.
func VerifyAP2Mandate(s AP2SignedMandate, public ed25519.PublicKey) error {
	sig, err := base64.RawURLEncoding.DecodeString(s.Signature)
	if err != nil {
		return fmt.Errorf("ap2: invalid signature encoding: %w", err)
	}
	payload, err := ap2Payload(s.Mandate)
	if err != nil {
		return err
	}
	if !ed25519.Verify(public, payload, sig) {
		return errors.New("ap2: mandate signature verification failed")
	}
	return nil
}

// AP2BindMandateToMessage returns a copy of m bound to the A2A message's task/context.
func AP2BindMandateToMessage(m AP2Mandate, msg Message) AP2Mandate {
	m.TaskID = msg.TaskID
	m.ContextID = msg.ContextID
	return m
}

// AP2AttachMandate returns a copy of msg carrying the signed AP2 mandate.
func AP2AttachMandate(msg Message, mandate AP2SignedMandate) Message {
	msg.AP2Mandates = append(append([]AP2SignedMandate{}, msg.AP2Mandates...), mandate)
	return msg
}

// VerifyAP2ForTask verifies signature, task binding, and optional settlement rail reference.
func VerifyAP2ForTask(s AP2SignedMandate, public ed25519.PublicKey, task Task, rail *AP2RailRef) AP2Verification {
	out := AP2Verification{MandateID: s.Mandate.ID, Kind: string(s.Mandate.Kind), Verified: true}
	if err := VerifyAP2Mandate(s, public); err != nil {
		out.Verified = false
		out.Error = err.Error()
		return out
	}
	if s.Mandate.TaskID != "" && s.Mandate.TaskID != task.ID {
		out.Verified = false
		out.Error = "ap2: mandate task binding mismatch"
		return out
	}
	if s.Mandate.ContextID != "" && s.Mandate.ContextID != task.ContextID {
		out.Verified = false
		out.Error = "ap2: mandate context binding mismatch"
		return out
	}
	if s.Mandate.Kind == AP2PaymentMandate && rail != nil {
		if s.Mandate.Rail == nil || *s.Mandate.Rail != *rail {
			out.Verified = false
			out.Error = "ap2: settlement rail reference mismatch"
			return out
		}
	}
	return out
}

// X402AP2Rail builds the x402 settlement rail reference carried under a payment mandate.
func X402AP2Rail(reference string) AP2RailRef { return AP2RailRef{Type: "x402", Reference: reference} }

func ap2Payload(m AP2Mandate) ([]byte, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("ap2: marshal mandate: %w", err)
	}
	sum := sha256.Sum256(b)
	return sum[:], nil
}
