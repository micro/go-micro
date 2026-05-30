package resource

import "testing"

func TestCommandsRegistered(t *testing.T) {
	// Each command func must return a command with a name and at least
	// one subcommand, so the resource surface stays consistent.
	for _, fn := range commandFuncs {
		c := fn()
		if c.Name == "" {
			t.Error("command with empty name")
		}
		if len(c.Subcommands) == 0 {
			t.Errorf("command %q has no subcommands", c.Name)
		}
	}
}

func TestExpectedCommands(t *testing.T) {
	names := map[string]bool{}
	for _, fn := range commandFuncs {
		names[fn().Name] = true
	}
	for _, want := range []string{"registry", "broker", "store", "config"} {
		if !names[want] {
			t.Errorf("missing %q command", want)
		}
	}
}
