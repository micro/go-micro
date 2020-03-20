package auth

import (
	"log"
	"testing"

	"github.com/micro/go-micro/v2/auth/token"

	"github.com/micro/go-micro/v2/auth/token/basic"
	memStore "github.com/micro/go-micro/v2/store/memory"
)

func TestGenerate(t *testing.T) {
	tokenProv := basic.NewTokenProvider(
		token.WithStore(memStore.NewStore()),
	)

	mem := NewAuth(
		TokenProvider(tokenProv),
		SecretProvider(tokenProv),
	)

	id := "test"
	roles := []string{"admin"}
	metadata := map[string]string{"foo": "bar"}

	opts := []GenerateOption{
		WithRoles(roles),
		WithMetadata(metadata),
	}

	// generate the account
	acc, err := mem.Generate(id, opts...)
	if err != nil {
		t.Fatalf("Generate returned an error: %v, expected nil", err)
	}
	// validate the account attributes were set correctly
	if acc.ID != id {
		t.Errorf("Generate returned %v as the ID, expected %v", acc.ID, id)
	}
	if len(acc.Roles) != len(roles) {
		t.Errorf("Generate returned %v as the roles, expected %v", acc.Roles, roles)
	}
	if len(acc.Metadata) != len(metadata) {
		t.Errorf("Generate returned %v as the metadata, expected %v", acc.Metadata, metadata)
	}

	// validate the token and secret are valid
	if tokenProv.Inspect(acc.Token.Token); err != nil {
		t.Errorf("Generate returned an invalid token, error: %v", err)
	}
	if tokenProv.Inspect(acc.Secret.Token); err != nil {
		t.Errorf("Generate returned an invalid secret, error: %v", err)
	}
}

func TestGrant(t *testing.T) {
	s := memStore.NewStore()
	a := NewAuth(Store(s))

	res := &Resource{Type: "service", Name: "Test", Endpoint: "Foo.Bar"}
	if err := a.Grant("users.*", res); err != nil {
		t.Fatalf("Grant returned an error: %v, expected nil", err)
	}

	recs, err := s.List()
	if err != nil {
		t.Fatalf("Could not read from the store: %v", err)
	}
	if len(recs) != 1 {
		t.Errorf("Expected Grant to write 1 record, actually wrote %v", len(recs))
	}
}

func TestRevoke(t *testing.T) {
	s := memStore.NewStore()
	a := NewAuth(Store(s))

	res := &Resource{Type: "service", Name: "Test", Endpoint: "Foo.Bar"}
	if err := a.Grant("users.*", res); err != nil {
		t.Fatalf("Grant returned an error: %v, expected nil", err)
	}

	recs, err := s.List()
	if err != nil {
		t.Fatalf("Could not read from the store: %v", err)
	}
	if len(recs) != 1 {
		t.Fatalf("Expected Grant to write 1 record, actually wrote %v", len(recs))
	}

	if err := a.Revoke("users.*", res); err != nil {
		t.Fatalf("Revoke returned an error: %v, expected nil", err)
	}

	recs, err = s.List()
	if err != nil {
		t.Fatalf("Could not read from the store: %v", err)
	}
	if len(recs) != 0 {
		t.Fatalf("Expected Revoke to delete 1 record, actually deleted %v", 1-len(recs))
	}
}

func TestInspect(t *testing.T) {
	a := NewAuth()

	t.Run("Valid Token", func(t *testing.T) {
		id := "test"
		roles := []string{"admin"}
		metadata := map[string]string{"foo": "bar"}

		opts := []GenerateOption{
			WithRoles(roles),
			WithMetadata(metadata),
		}

		// generate and inspect the token
		tok, err := a.Generate("test", opts...)
		if err != nil {
			log.Fatalf("Generate returned an error: %v, expected nil", err)
		}
		acc, err := a.Inspect(tok.Token.Token)
		if err != nil {
			log.Fatalf("Inspect returned an error: %v, expected nil", err)
		}

		// validate the account attributes were retrieved correctly
		if acc.ID != id {
			t.Errorf("Generate returned %v as the ID, expected %v", acc.ID, id)
		}
		if len(acc.Roles) != len(roles) {
			t.Errorf("Generate returned %v as the roles, expected %v", acc.Roles, roles)
		}
		if len(acc.Metadata) != len(metadata) {
			t.Errorf("Generate returned %v as the metadata, expected %v", acc.Metadata, metadata)
		}
	})

	t.Run("Invalid Token", func(t *testing.T) {
		_, err := a.Inspect("invalid token")
		if err != ErrInvalidToken {
			t.Errorf("Inspect returned %v error, expected %v", err, ErrInvalidToken)
		}
	})
}

func TestRefresh(t *testing.T) {
	a := NewAuth()

	t.Run("Valid Secret", func(t *testing.T) {
		roles := []string{"admin"}
		metadata := map[string]string{"foo": "bar"}

		opts := []GenerateOption{
			WithRoles(roles),
			WithMetadata(metadata),
		}

		// generate the account
		acc, err := a.Generate("test", opts...)
		if err != nil {
			log.Fatalf("Generate returned an error: %v, expected nil", err)
		}

		// refresh the token
		tok, err := a.Refresh(acc.Secret.Token)
		if err != nil {
			log.Fatalf("Refresh returned an error: %v, expected nil", err)
		}

		// validate the account attributes were set correctly
		if acc.ID != tok.Subject {
			t.Errorf("Refresh returned %v as the ID, expected %v", acc.ID, tok.Subject)
		}
		if len(acc.Roles) != len(tok.Roles) {
			t.Errorf("Refresh returned %v as the roles, expected %v", acc.Roles, tok.Subject)
		}
		if len(acc.Metadata) != len(tok.Metadata) {
			t.Errorf("Refresh returned %v as the metadata, expected %v", acc.Metadata, tok.Metadata)
		}
	})

	t.Run("Invalid Secret", func(t *testing.T) {
		_, err := a.Refresh("invalid secret")
		if err != ErrInvalidToken {
			t.Errorf("Inspect returned %v error, expected %v", err, ErrInvalidToken)
		}
	})
}

func TestVerify(t *testing.T) {
	testRules := []struct {
		Role     string
		Resource *Resource
	}{
		{
			Role:     "*",
			Resource: &Resource{Type: "service", Name: "go.micro.apps", Endpoint: "Apps.PublicList"},
		},
		{
			Role:     "user.*",
			Resource: &Resource{Type: "service", Name: "go.micro.apps", Endpoint: "Apps.List"},
		},
		{
			Role:     "user.developer",
			Resource: &Resource{Type: "service", Name: "go.micro.apps", Endpoint: "Apps.Update"},
		},
		{
			Role:     "admin",
			Resource: &Resource{Type: "service", Name: "go.micro.apps", Endpoint: "Apps.Delete"},
		},
		{
			Role:     "admin",
			Resource: &Resource{Type: "service", Name: "*", Endpoint: "*"},
		},
	}

	a := NewAuth()
	for _, r := range testRules {
		if err := a.Grant(r.Role, r.Resource); err != nil {
			t.Fatalf("Grant returned an error: %v, expected nil", err)
		}
	}

	testTable := []struct {
		Name     string
		Roles    []string
		Resource *Resource
		Error    error
	}{
		{
			Name:     "An account with no roles accessing a public endpoint",
			Resource: &Resource{Type: "service", Name: "go.micro.apps", Endpoint: "Apps.PublicList"},
		},
		{
			Name:     "An account with no roles accessing a private endpoint",
			Resource: &Resource{Type: "service", Name: "go.micro.apps", Endpoint: "Apps.Update"},
			Error:    ErrForbidden,
		},
		{
			Name:     "An account with the user role accessing a user* endpoint",
			Roles:    []string{"user"},
			Resource: &Resource{Type: "service", Name: "go.micro.apps", Endpoint: "Apps.List"},
		},
		{
			Name:     "An account with the user role accessing a user.admin endpoint",
			Roles:    []string{"user"},
			Resource: &Resource{Type: "service", Name: "go.micro.apps", Endpoint: "Apps.Delete"},
			Error:    ErrForbidden,
		},
		{
			Name:     "An account with the developer role accessing a user.developer endpoint",
			Roles:    []string{"user.developer"},
			Resource: &Resource{Type: "service", Name: "go.micro.apps", Endpoint: "Apps.Update"},
		},
		{
			Name:     "An account with the developer role accessing an admin endpoint",
			Roles:    []string{"user.developer"},
			Resource: &Resource{Type: "service", Name: "go.micro.apps", Endpoint: "Apps.Delete"},
			Error:    ErrForbidden,
		},
		{
			Name:     "An admin account accessing an admin endpoint",
			Roles:    []string{"admin"},
			Resource: &Resource{Type: "service", Name: "go.micro.apps", Endpoint: "Apps.Delete"},
		},
		{
			Name:     "An admin account accessing a generic service endpoint",
			Roles:    []string{"admin"},
			Resource: &Resource{Type: "service", Name: "go.micro.foo", Endpoint: "Foo.Bar"},
		},
		{
			Name:     "An admin account accessing an unauthorised endpoint",
			Roles:    []string{"admin"},
			Resource: &Resource{Type: "infra", Name: "go.micro.foo", Endpoint: "Foo.Bar"},
			Error:    ErrForbidden,
		},
		{
			Name:     "A account with no roles accessing an unauthorised endpoint",
			Resource: &Resource{Type: "infra", Name: "go.micro.foo", Endpoint: "Foo.Bar"},
			Error:    ErrForbidden,
		},
	}

	for _, tc := range testTable {
		t.Run(tc.Name, func(t *testing.T) {
			acc := &Account{Roles: tc.Roles}
			if err := a.Verify(acc, tc.Resource); err != tc.Error {
				t.Errorf("Verify returned %v error, expected %v", err, tc.Error)
			}
		})
	}
}
