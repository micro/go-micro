package auth

import "testing"

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
