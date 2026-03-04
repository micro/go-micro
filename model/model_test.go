package model

import (
	"reflect"
	"testing"
)

type TestUser struct {
	ID    string `json:"id" model:"key"`
	Name  string `json:"name" model:"index"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

func TestBuildSchema(t *testing.T) {
	schema := buildSchema(reflect.TypeOf(TestUser{}))

	if schema.Table != "testusers" {
		t.Errorf("expected table 'testusers', got %q", schema.Table)
	}
	if schema.Key != "id" {
		t.Errorf("expected key 'id', got %q", schema.Key)
	}
	if len(schema.Fields) != 4 {
		t.Fatalf("expected 4 fields, got %d", len(schema.Fields))
	}

	// Check key field
	var keyField Field
	var indexField Field
	for _, f := range schema.Fields {
		if f.IsKey {
			keyField = f
		}
		if f.Index {
			indexField = f
		}
	}
	if keyField.Column != "id" {
		t.Errorf("expected key column 'id', got %q", keyField.Column)
	}
	if indexField.Column != "name" {
		t.Errorf("expected index column 'name', got %q", indexField.Column)
	}
}

func TestBuildSchema_DefaultKey(t *testing.T) {
	type Item struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	schema := buildSchema(reflect.TypeOf(Item{}))
	if schema.Key != "id" {
		t.Errorf("expected default key 'id', got %q", schema.Key)
	}
}

func TestBuildSchema_WithTable(t *testing.T) {
	schema := buildSchema(reflect.TypeOf(TestUser{}))
	WithTable("my_users")(schema)

	if schema.Table != "my_users" {
		t.Errorf("expected table 'my_users', got %q", schema.Table)
	}
}

func TestStructToMap(t *testing.T) {
	schema := buildSchema(reflect.TypeOf(TestUser{}))
	u := &TestUser{ID: "1", Name: "Alice", Email: "alice@example.com", Age: 30}

	m := structToMap(schema, u)

	if m["id"] != "1" {
		t.Errorf("expected id '1', got %v", m["id"])
	}
	if m["name"] != "Alice" {
		t.Errorf("expected name 'Alice', got %v", m["name"])
	}
	if m["email"] != "alice@example.com" {
		t.Errorf("expected email 'alice@example.com', got %v", m["email"])
	}
	if m["age"] != 30 {
		t.Errorf("expected age 30, got %v", m["age"])
	}
}

func TestMapToStruct(t *testing.T) {
	schema := buildSchema(reflect.TypeOf(TestUser{}))
	m := map[string]any{
		"id":    "1",
		"name":  "Bob",
		"email": "bob@example.com",
		"age":   25,
	}

	u := mapToStruct[TestUser](schema, m)

	if u.ID != "1" {
		t.Errorf("expected ID '1', got %q", u.ID)
	}
	if u.Name != "Bob" {
		t.Errorf("expected Name 'Bob', got %q", u.Name)
	}
	if u.Email != "bob@example.com" {
		t.Errorf("expected Email 'bob@example.com', got %q", u.Email)
	}
	if u.Age != 25 {
		t.Errorf("expected Age 25, got %d", u.Age)
	}
}

func TestApplyQueryOptions(t *testing.T) {
	q := ApplyQueryOptions(
		Where("name", "Alice"),
		WhereOp("age", ">", 20),
		OrderDesc("name"),
		Limit(10),
		Offset(5),
	)

	if len(q.Filters) != 2 {
		t.Fatalf("expected 2 filters, got %d", len(q.Filters))
	}
	if q.Filters[0].Field != "name" || q.Filters[0].Op != "=" || q.Filters[0].Value != "Alice" {
		t.Errorf("unexpected filter 0: %+v", q.Filters[0])
	}
	if q.Filters[1].Field != "age" || q.Filters[1].Op != ">" {
		t.Errorf("unexpected filter 1: %+v", q.Filters[1])
	}
	if q.OrderBy != "name" || !q.Desc {
		t.Errorf("expected order by name desc, got %q desc=%v", q.OrderBy, q.Desc)
	}
	if q.Limit != 10 {
		t.Errorf("expected limit 10, got %d", q.Limit)
	}
	if q.Offset != 5 {
		t.Errorf("expected offset 5, got %d", q.Offset)
	}
}
