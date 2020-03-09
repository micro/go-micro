package cache

import "testing"

func TestCache(t *testing.T) {
	c := NewStore()
	if err := c.Init(); err != nil {
		t.Fatal(err)
	}
	if results, err := c.Read("test"); err != nil {
		t.Fatal(err)
	} else {
		println(results)
	}
}
