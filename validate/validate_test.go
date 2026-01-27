package validate

import (
	"reflect"
	"testing"
)

type User struct {
	Name     string `json:"name" validate:"required,min=2,max=50"`
	Email    string `json:"email" validate:"required,email"`
	Age      int    `json:"age" validate:"min=0,max=150"`
	Website  string `json:"website" validate:"url"`
	Username string `json:"username" validate:"alphanum"`
	ID       string `json:"id" validate:"uuid"`
}

func TestValidate_Required(t *testing.T) {
	user := User{Name: "", Email: "test@example.com"}
	errs := Validate(&user)

	if !errs.HasErrors() {
		t.Error("expected validation errors")
	}

	found := false
	for _, err := range errs {
		if err.Field == "name" && err.Tag == "required" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'name required' error")
	}
}

func TestValidate_Email(t *testing.T) {
	user := User{Name: "John", Email: "invalid"}
	errs := Validate(&user)

	found := false
	for _, err := range errs {
		if err.Field == "email" && err.Tag == "email" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'email' validation error")
	}
}

func TestValidate_MinMax(t *testing.T) {
	user := User{Name: "J", Email: "test@example.com"}
	errs := Validate(&user)

	found := false
	for _, err := range errs {
		if err.Field == "name" && err.Tag == "min" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'name min' error")
	}
}

func TestValidate_Valid(t *testing.T) {
	user := User{
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   30,
	}
	errs := Validate(&user)

	// Filter out optional field errors (website, username, id)
	var requiredErrors ValidationErrors
	for _, err := range errs {
		if err.Field == "name" || err.Field == "email" || err.Field == "age" {
			requiredErrors = append(requiredErrors, err)
		}
	}

	if requiredErrors.HasErrors() {
		t.Errorf("expected no errors for required fields, got: %v", requiredErrors)
	}
}

func TestValidate_UUID(t *testing.T) {
	user := User{
		Name:  "John",
		Email: "john@example.com",
		ID:    "not-a-uuid",
	}
	errs := Validate(&user)

	found := false
	for _, err := range errs {
		if err.Field == "id" && err.Tag == "uuid" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'uuid' validation error")
	}

	// Test valid UUID
	user.ID = "550e8400-e29b-41d4-a716-446655440000"
	errs = Validate(&user)
	found = false
	for _, err := range errs {
		if err.Field == "id" && err.Tag == "uuid" {
			found = true
			break
		}
	}
	if found {
		t.Error("should not have uuid error for valid UUID")
	}
}

func TestValidate_CustomValidator(t *testing.T) {
	v := New()
	v.RegisterValidator("even", func(field reflect.Value, _ string) bool {
		if field.Kind() != reflect.Int {
			return false
		}
		return field.Int()%2 == 0
	})

	type Number struct {
		Value int `validate:"even"`
	}

	errs := v.Validate(&Number{Value: 3})
	if !errs.HasErrors() {
		t.Error("expected error for odd number")
	}

	errs = v.Validate(&Number{Value: 4})
	if errs.HasErrors() {
		t.Error("expected no error for even number")
	}
}
