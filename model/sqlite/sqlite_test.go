package sqlite

import (
	"context"
	"testing"

	"go-micro.dev/v5/model"
)

type User struct {
	ID    string `json:"id" model:"key"`
	Name  string `json:"name" model:"index"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

func setup(t *testing.T) model.Model {
	t.Helper()
	db := New(":memory:")
	if err := db.Register(&User{}); err != nil {
		t.Fatalf("register: %v", err)
	}
	return db
}

func TestCRUD(t *testing.T) {
	db := setup(t)
	ctx := context.Background()

	// Create
	err := db.Create(ctx, &User{ID: "1", Name: "Alice", Email: "alice@test.com", Age: 30})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Read
	u := &User{}
	err = db.Read(ctx, "1", u)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if u.Name != "Alice" {
		t.Errorf("expected Alice, got %s", u.Name)
	}
	if u.Age != 30 {
		t.Errorf("expected age 30, got %d", u.Age)
	}

	// Update
	u.Name = "Alice Updated"
	u.Age = 31
	err = db.Update(ctx, u)
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	u2 := &User{}
	db.Read(ctx, "1", u2)
	if u2.Name != "Alice Updated" {
		t.Errorf("expected 'Alice Updated', got %s", u2.Name)
	}
	if u2.Age != 31 {
		t.Errorf("expected age 31, got %d", u2.Age)
	}

	// Delete
	err = db.Delete(ctx, "1", &User{})
	if err != nil {
		t.Fatalf("delete: %v", err)
	}

	err = db.Read(ctx, "1", &User{})
	if err != model.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDuplicateKey(t *testing.T) {
	db := setup(t)
	ctx := context.Background()

	db.Create(ctx, &User{ID: "1", Name: "Alice"})
	err := db.Create(ctx, &User{ID: "1", Name: "Bob"})
	if err != model.ErrDuplicateKey {
		t.Errorf("expected ErrDuplicateKey, got %v", err)
	}
}

func TestNotFound(t *testing.T) {
	db := setup(t)
	ctx := context.Background()

	err := db.Read(ctx, "nonexistent", &User{})
	if err != model.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}

	err = db.Update(ctx, &User{ID: "nonexistent"})
	if err != model.ErrNotFound {
		t.Errorf("expected ErrNotFound on update, got %v", err)
	}

	err = db.Delete(ctx, "nonexistent", &User{})
	if err != model.ErrNotFound {
		t.Errorf("expected ErrNotFound on delete, got %v", err)
	}
}

func TestListWithFilter(t *testing.T) {
	db := setup(t)
	ctx := context.Background()

	db.Create(ctx, &User{ID: "1", Name: "Alice", Age: 30})
	db.Create(ctx, &User{ID: "2", Name: "Bob", Age: 25})
	db.Create(ctx, &User{ID: "3", Name: "Alice", Age: 35})

	var results []*User
	err := db.List(ctx, &results, model.Where("name", "Alice"))
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 Alices, got %d", len(results))
	}
}

func TestListWithOrder(t *testing.T) {
	db := setup(t)
	ctx := context.Background()

	db.Create(ctx, &User{ID: "1", Name: "Charlie", Age: 35})
	db.Create(ctx, &User{ID: "2", Name: "Alice", Age: 30})
	db.Create(ctx, &User{ID: "3", Name: "Bob", Age: 25})

	var results []*User
	err := db.List(ctx, &results, model.OrderAsc("name"))
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3, got %d", len(results))
	}
	if results[0].Name != "Alice" {
		t.Errorf("expected Alice first, got %s", results[0].Name)
	}
	if results[2].Name != "Charlie" {
		t.Errorf("expected Charlie last, got %s", results[2].Name)
	}
}

func TestListWithLimitOffset(t *testing.T) {
	db := setup(t)
	ctx := context.Background()

	db.Create(ctx, &User{ID: "1", Name: "A", Age: 1})
	db.Create(ctx, &User{ID: "2", Name: "B", Age: 2})
	db.Create(ctx, &User{ID: "3", Name: "C", Age: 3})
	db.Create(ctx, &User{ID: "4", Name: "D", Age: 4})

	var results []*User
	err := db.List(ctx, &results,
		model.OrderAsc("name"),
		model.Limit(2),
		model.Offset(1),
	)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2, got %d", len(results))
	}
	if results[0].Name != "B" {
		t.Errorf("expected B, got %s", results[0].Name)
	}
	if results[1].Name != "C" {
		t.Errorf("expected C, got %s", results[1].Name)
	}
}

func TestCount(t *testing.T) {
	db := setup(t)
	ctx := context.Background()

	db.Create(ctx, &User{ID: "1", Name: "Alice", Age: 30})
	db.Create(ctx, &User{ID: "2", Name: "Bob", Age: 25})
	db.Create(ctx, &User{ID: "3", Name: "Alice", Age: 35})

	count, err := db.Count(ctx, &User{})
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3, got %d", count)
	}

	count, err = db.Count(ctx, &User{}, model.Where("name", "Alice"))
	if err != nil {
		t.Fatalf("count with filter: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}

func TestWhereOp(t *testing.T) {
	db := setup(t)
	ctx := context.Background()

	db.Create(ctx, &User{ID: "1", Name: "Alice", Age: 30})
	db.Create(ctx, &User{ID: "2", Name: "Bob", Age: 25})
	db.Create(ctx, &User{ID: "3", Name: "Charlie", Age: 35})

	var results []*User
	err := db.List(ctx, &results, model.WhereOp("age", ">", 28))
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 (age > 28), got %d", len(results))
	}
}
