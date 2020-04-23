package qson

import "testing"

func TestMergeSlice(t *testing.T) {
	a := []interface{}{"a"}
	b := []interface{}{"b"}
	actual := mergeSlice(a, b)
	if len(actual) != 2 {
		t.Errorf("Expected size to be 2.")
	}
	if actual[0] != "a" {
		t.Errorf("Expected index 0 to have value a. Actual: %s", actual[0])
	}
	if actual[1] != "b" {
		t.Errorf("Expected index 1 to have value b. Actual: %s", actual[1])
	}
}

func TestMergeMap(t *testing.T) {
	a := map[string]interface{}{
		"a": "b",
	}
	b := map[string]interface{}{
		"b": "c",
	}
	actual := mergeMap(a, b)
	if len(actual) != 2 {
		t.Errorf("Expected size to be 2.")
	}
	if actual["a"] != "b" {
		t.Errorf("Expected key \"a\" to have value b. Actual: %s", actual["a"])
	}
	if actual["b"] != "c" {
		t.Errorf("Expected key \"b\" to have value c. Actual: %s", actual["b"])
	}
}
