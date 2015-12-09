package blacklist

import (
	"errors"
	"testing"
	"time"

	"github.com/micro/go-micro/registry/mock"
	"github.com/micro/go-micro/selector"
)

func TestBlackListSelector(t *testing.T) {
	counts := map[string]int{}

	bl := &blackListSelector{
		so: selector.Options{
			Registry: mock.NewRegistry(),
		},
		ttl:  2,
		bl:   make(map[string]blackListNode),
		exit: make(chan bool),
	}

	go bl.run()
	defer bl.Close()

	next, err := bl.Select("foo")
	if err != nil {
		t.Errorf("Unexpected error calling bl select: %v", err)
	}

	for i := 0; i < 100; i++ {
		node, err := next()
		if err != nil {
			t.Errorf("Expected node err, got err: %v", err)
		}
		counts[node.Id]++
	}

	t.Logf("BlackList Counts %v", counts)

	// test blacklisting
	for i := 0; i < 4; i++ {
		node, err := next()
		if err != nil {
			t.Errorf("Expected node err, got err: %v", err)
		}
		bl.Mark("foo", node, errors.New("blacklist"))
	}
	if node, err := next(); err != selector.ErrNoneAvailable {
		t.Errorf("Expected none available err, got node %v err %v", node, err)
	}
	time.Sleep(time.Second * time.Duration(bl.ttl) * 2)
	if _, err := next(); err != nil {
		t.Errorf("Unexpected err %v", err)
	}

	// test resetting
	for i := 0; i < 4; i++ {
		node, err := next()
		if err != nil {
			t.Errorf("Unexpected err: %v", err)
		}
		bl.Mark("foo", node, errors.New("blacklist"))
	}
	if node, err := next(); err != selector.ErrNoneAvailable {
		t.Errorf("Expected none available err, got node %v err %v", node, err)
	}
	bl.Reset("foo")
	if _, err := next(); err != nil {
		t.Errorf("Unexpected err %v", err)
	}
}
