package ring

import (
	"testing"
	"time"
)

func TestBuffer(t *testing.T) {
	b := New(10)

	// test one value
	b.Put("foo")
	v := b.Get(1)

	if val := v[0].Value.(string); val != "foo" {
		t.Fatalf("expected foo got %v", val)
	}

	b = New(10)

	// test 10 values
	for i := 0; i < 10; i++ {
		b.Put(i)
	}

	d := time.Now()
	v = b.Get(10)

	for i := 0; i < 10; i++ {
		val := v[i].Value.(int)

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
		val := v[i].Value.(int)
		expect := i * 2
		if val != expect {
			t.Fatalf("expected %d got %d", expect, val)
		}
	}

	// sleep 100 ms
	time.Sleep(time.Millisecond * 100)

	// assume we'll get everything
	v = b.Since(d)

	if len(v) != 10 {
		t.Fatalf("expected 10 entries but got %d", len(v))
	}

	// write 1 more entry
	d = time.Now()
	b.Put(100)

	// sleep 100 ms
	time.Sleep(time.Millisecond * 100)

	v = b.Since(d)
	if len(v) != 1 {
		t.Fatalf("expected 1 entries but got %d", len(v))
	}

	if v[0].Value.(int) != 100 {
		t.Fatalf("expected value 100 got %v", v[0])
	}
}
