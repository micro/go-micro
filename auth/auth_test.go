package auth

import "testing"

func TestHasScope(t *testing.T) {
	if new(Account).HasScope("namespace", "foo") {
		t.Errorf("Expected the blank account to not have a role")
	}

	acc := Account{Scopes: []string{"namespace.foo"}}
	if !acc.HasScope("namespace", "foo") {
		t.Errorf("Expected the account to have the namespace.foo role")
	}
	if acc.HasScope("namespace", "bar") {
		t.Errorf("Expected the account to not have the namespace.bar role")
	}
}
func TestHasRole(t *testing.T) {
	if new(Account).HasRole("foo") {
		t.Errorf("Expected the blank account to not have a role")
	}

	acc := Account{Roles: []string{"foo"}}
	if !acc.HasRole("foo") {
		t.Errorf("Expected the account to have the foo role")
	}
	if acc.HasRole("bar") {
		t.Errorf("Expected the account to not have the bar role")
	}
}
