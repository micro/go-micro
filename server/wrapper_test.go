package server

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

var (
	errBefore  = errors.New("before hook failed")
	errAfter   = errors.New("after hook failed")
	errHandler = errors.New("handler failed")
)

func TestBeforeHandler(t *testing.T) {
	var order []string

	before := func(ctx context.Context, req Request) error {
		order = append(order, "before")
		return nil
	}
	handler := func(ctx context.Context, req Request, rsp interface{}) error {
		order = append(order, "handler")
		return nil
	}

	h := BeforeHandler(before)(handler)
	if err := h(context.Background(), nil, nil); err != nil {
		t.Fatalf("BeforeHandler returned error: %v", err)
	}

	want := []string{"before", "handler"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("order = %v, want %v", order, want)
	}
}

func TestBeforeHandlerAbortsOnError(t *testing.T) {
	called := false

	before := func(ctx context.Context, req Request) error {
		return errBefore
	}
	handler := func(ctx context.Context, req Request, rsp interface{}) error {
		called = true
		return nil
	}

	h := BeforeHandler(before)(handler)
	if err := h(context.Background(), nil, nil); err != errBefore {
		t.Fatalf("err = %v, want %v", err, errBefore)
	}
	if called {
		t.Fatal("handler invoked despite before hook error")
	}
}

func TestAfterHandler(t *testing.T) {
	var order []string

	handler := func(ctx context.Context, req Request, rsp interface{}) error {
		order = append(order, "handler")
		return nil
	}
	after := func(ctx context.Context, req Request) error {
		order = append(order, "after")
		return nil
	}

	h := AfterHandler(after)(handler)
	if err := h(context.Background(), nil, nil); err != nil {
		t.Fatalf("AfterHandler returned error: %v", err)
	}

	want := []string{"handler", "after"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("order = %v, want %v", order, want)
	}
}

func TestAfterHandlerRunsAfterHandlerError(t *testing.T) {
	var order []string

	handler := func(ctx context.Context, req Request, rsp interface{}) error {
		order = append(order, "handler")
		return errHandler
	}
	after := func(ctx context.Context, req Request) error {
		order = append(order, "after")
		return nil
	}

	h := AfterHandler(after)(handler)
	if err := h(context.Background(), nil, nil); err != errHandler {
		t.Fatalf("err = %v, want handler error %v", err, errHandler)
	}

	want := []string{"handler", "after"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("order = %v, want %v", order, want)
	}
}

func TestAfterHandlerErrorReturnedWhenHandlerSucceeds(t *testing.T) {
	handler := func(ctx context.Context, req Request, rsp interface{}) error {
		return nil
	}
	after := func(ctx context.Context, req Request) error {
		return errAfter
	}

	h := AfterHandler(after)(handler)
	if err := h(context.Background(), nil, nil); err != errAfter {
		t.Fatalf("err = %v, want %v", err, errAfter)
	}
}

func TestBeforeAndAfterHandlerCompose(t *testing.T) {
	var order []string

	before := func(ctx context.Context, req Request) error {
		order = append(order, "before")
		return nil
	}
	handler := func(ctx context.Context, req Request, rsp interface{}) error {
		order = append(order, "handler")
		return nil
	}
	after := func(ctx context.Context, req Request) error {
		order = append(order, "after")
		return nil
	}

	h := BeforeHandler(before)(AfterHandler(after)(handler))
	if err := h(context.Background(), nil, nil); err != nil {
		t.Fatalf("composed handler returned error: %v", err)
	}

	want := []string{"before", "handler", "after"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("order = %v, want %v", order, want)
	}
}
