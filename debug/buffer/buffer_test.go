package buffer

import (
	"testing"
)

func TestBuffer(t *testing.T) {
	b := New(10)

	// test one value
	b.Put("foo")
	v := b.Get(1)

	if val := v[0].(string); val != "foo" {
		t.Fatalf("expected foo got %v", val)
	}

	b = New(10)

	// test 10 values
	for i := 0; i < 10; i++ {
		b.Put(i)
	}

	v = b.Get(10)

	for i := 0; i < 10; i++ {
		val := v[i].(int)

		if val != i {
			t.Fatalf("expected %d got %d", i, val)
		}
	}

	// test more values

	for i := 0; i < 10; i++ {
		v := i * 2
		b.Put(v)
	}

	v = b.Get(10)

	for i := 0; i < 10; i++ {
		val := v[i].(int)
		expect := i * 2
		if val != expect {
			t.Fatalf("expected %d got %d", expect, val)
		}
	}

}
