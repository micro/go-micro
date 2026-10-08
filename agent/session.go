package agent

import (
	"context"
	"fmt"

	"go-micro.dev/v6/metadata"
	"go-micro.dev/v6/store"
)

// WithSession selects a conversation when calling an agent over RPC. The host
// must authorize access to the agent; a session ID is not an access credential.
func WithSession(ctx context.Context, id string) context.Context {
	return metadata.Set(ctx, "Micro-Session", id)
}

// RPC conversations are loaded from the existing store for each request. Keep
// requests serialized so two clients cannot overwrite the same history.
func (a *agentImpl) rpcSession(ctx context.Context) (*agentImpl, func(), error) {
	id, _ := metadata.Get(ctx, "Micro-Session")
	if id == "" {
		return a, func() {}, nil
	}
	if len(id) > 128 {
		return nil, nil, fmt.Errorf("agent: session ID exceeds 128 bytes")
	}
	if a.opts.Memory != nil {
		return nil, nil, fmt.Errorf("agent: RPC sessions require store-backed memory")
	}
	a.sessionOnce.Do(func() { a.sessionGate = make(chan struct{}, 1) })
	select {
	case a.sessionGate <- struct{}{}:
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	}
	release := func() { <-a.sessionGate }
	opts := a.opts
	opts.Session = id
	child := &agentImpl{opts: opts}
	if _, err := child.stateStore().List(store.ListLimit(1)); err != nil {
		release()
		return nil, nil, fmt.Errorf("agent session storage: %w", err)
	}
	return child, release, nil
}
