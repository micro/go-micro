package x402

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestLiveFacilitatorConformance is an opt-in smoke test for hosted x402
// facilitators. It is skipped by default so local and CI runs never need live
// credentials, but operators can run it against Coinbase, Alchemy, or a
// self-hosted facilitator by providing a real payment payload and matching
// requirements.
func TestLiveFacilitatorConformance(t *testing.T) {
	url := os.Getenv("GO_MICRO_X402_LIVE_FACILITATOR_URL")
	payment := os.Getenv("GO_MICRO_X402_LIVE_PAYMENT")
	payTo := os.Getenv("GO_MICRO_X402_LIVE_PAY_TO")
	if url == "" || payment == "" || payTo == "" {
		t.Skip("set GO_MICRO_X402_LIVE_FACILITATOR_URL, GO_MICRO_X402_LIVE_PAYMENT, and GO_MICRO_X402_LIVE_PAY_TO to run live x402 facilitator conformance")
	}

	network := getenv("GO_MICRO_X402_LIVE_NETWORK", "base")
	amount := getenv("GO_MICRO_X402_LIVE_AMOUNT", "1")
	asset := os.Getenv("GO_MICRO_X402_LIVE_ASSET")
	resource := getenv("GO_MICRO_X402_LIVE_RESOURCE", "go-micro-x402-live-conformance")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	res, err := (&HTTPFacilitator{URL: url}).Verify(ctx, payment, Requirements{
		Scheme:            "exact",
		Network:           network,
		MaxAmountRequired: amount,
		Resource:          resource,
		PayTo:             payTo,
		Asset:             asset,
		MaxTimeoutSeconds: 60,
	})
	if err != nil {
		t.Fatalf("live facilitator verify: %v", err)
	}
	if !res.Valid {
		t.Fatalf("live facilitator rejected payment: %s", res.Reason)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
