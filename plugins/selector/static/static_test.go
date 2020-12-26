package static

import (
	"fmt"
	"os"
	"testing"
)

const (
	TEST_DOMAIN_NAME = "micro.mu"
	TEST_PORT_NUMBER = "3333"
)

func TestStaticSelectorWithDefaults(t *testing.T) {
	data := []string{"foo", "bar", "baz"}

	// Make sure both env-vars are empty (set to default):
	os.Setenv(ENV_STATIC_SELECTOR_DOMAIN_NAME, "")
	os.Setenv(ENV_STATIC_SELECTOR_PORT_NUMBER, "")

	s := NewSelector()

	for _, name := range data {
		next, err := s.Select(name)
		if err != nil {
			t.Fatal(err)
		}

		for i := 0; i < 3; i++ {
			node, err := next()
			if err != nil {
				t.Fatal(err)
			}

			expectedAddress := fmt.Sprintf("%v:%v", name, DEFAULT_PORT_NUMBER)

			if node.Address != expectedAddress {
				t.Fatalf("got %s expected %s", node.Address, expectedAddress)
			}
		}
	}
}

func TestStaticSelectorWithDomainNameOverride(t *testing.T) {
	data := []string{"foo", "bar", "baz"}

	// Make sure both env-vars are correctly set:
	os.Setenv(ENV_STATIC_SELECTOR_DOMAIN_NAME, TEST_DOMAIN_NAME)
	os.Setenv(ENV_STATIC_SELECTOR_PORT_NUMBER, "")

	s := NewSelector()

	for _, name := range data {
		next, err := s.Select(name)
		if err != nil {
			t.Fatal(err)
		}

		for i := 0; i < 3; i++ {
			node, err := next()
			if err != nil {
				t.Fatal(err)
			}

			expectedAddress := fmt.Sprintf("%v.%v:%v", name, TEST_DOMAIN_NAME, DEFAULT_PORT_NUMBER)

			if node.Address != expectedAddress {
				t.Fatalf("got %s expected %s", node.Address, expectedAddress)
			}
		}
	}
}

func TestStaticSelectorWithPortNumberOverride(t *testing.T) {
	data := []string{"foo", "bar", "baz"}

	// Make sure both env-vars are correctly set:
	os.Setenv(ENV_STATIC_SELECTOR_DOMAIN_NAME, "")
	os.Setenv(ENV_STATIC_SELECTOR_PORT_NUMBER, TEST_PORT_NUMBER)

	s := NewSelector()

	for _, name := range data {
		next, err := s.Select(name)
		if err != nil {
			t.Fatal(err)
		}

		for i := 0; i < 3; i++ {
			node, err := next()
			if err != nil {
				t.Fatal(err)
			}

			expectedAddress := fmt.Sprintf("%v:%v", name, TEST_PORT_NUMBER)

			if node.Address != expectedAddress {
				t.Fatalf("got %s expected %s", node.Address, expectedAddress)
			}
		}
	}
}

func TestStaticSelectorWithBothOverrides(t *testing.T) {
	data := []string{"foo", "bar", "baz"}

	// Make sure both env-vars are correctly set:
	os.Setenv(ENV_STATIC_SELECTOR_DOMAIN_NAME, TEST_DOMAIN_NAME)
	os.Setenv(ENV_STATIC_SELECTOR_PORT_NUMBER, TEST_PORT_NUMBER)

	s := NewSelector()

	for _, name := range data {
		next, err := s.Select(name)
		if err != nil {
			t.Fatal(err)
		}

		for i := 0; i < 3; i++ {
			node, err := next()
			if err != nil {
				t.Fatal(err)
			}

			expectedAddress := fmt.Sprintf("%v.%v:%v", name, TEST_DOMAIN_NAME, TEST_PORT_NUMBER)

			if node.Address != expectedAddress {
				t.Fatalf("got %s expected %s", node.Address, expectedAddress)
			}
		}
	}
}
