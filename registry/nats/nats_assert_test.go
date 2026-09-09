package nats_test

import (
	"reflect"
	"testing"
)

func assertNoError(tb testing.TB, actual error) {
	if actual != nil {
		tb.Errorf("expected no error, got %v", actual)
	}
}

func assertEqual(tb testing.TB, expected, actual interface{}) {
	if !reflect.DeepEqual(expected, actual) {
		tb.Errorf("expected %v, got %v", expected, actual)
	}
}
