package command

import (
	"testing"
)

func TestCommand(t *testing.T) {
	c := &cmd{
		name:        "test",
		usage:       "test usage",
		description: "test description",
		exec: func(args ...string) ([]byte, error) {
			return []byte("test"), nil
		},
	}

	if c.String() != c.name {
		t.Fatalf("expected name %s got %s", c.name, c.String())
	}

	if c.Usage() != c.usage {
		t.Fatalf("expected usage %s got %s", c.usage, c.Usage())
	}

	if c.Description() != c.description {
		t.Fatalf("expected description %s got %s", c.description, c.Description())
	}

	if r, err := c.Exec(); err != nil {
		t.Fatal(err)
	} else if string(r) != "test" {
		t.Fatalf("expected exec result test got %s", string(r))
	}
}

func TestNewCommand(t *testing.T) {
	c := &cmd{
		name:        "test",
		usage:       "test usage",
		description: "test description",
		exec: func(args ...string) ([]byte, error) {
			return []byte("test"), nil
		},
	}

	nc := NewCommand(c.name, c.usage, c.description, c.exec)

	if nc.String() != c.name {
		t.Fatalf("expected name %s got %s", c.name, nc.String())
	}

	if nc.Usage() != c.usage {
		t.Fatalf("expected usage %s got %s", c.usage, nc.Usage())
	}

	if nc.Description() != c.description {
		t.Fatalf("expected description %s got %s", c.description, nc.Description())
	}

	if r, err := nc.Exec(); err != nil {
		t.Fatal(err)
	} else if string(r) != "test" {
		t.Fatalf("expected exec result test got %s", string(r))
	}
}
