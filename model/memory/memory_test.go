package memory

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

func setup(t *testing.T) *model.Model[User] {
	t.Helper()
	db := New()
	return model.New[User](db)
}

func TestCRUD(t *testing.T) {
	users := setup(t)
	ctx := context.Background()

	// Create
	err := users.Create(ctx, &User{ID: "1", Name: "Alice", Email: "alice@test.com", Age: 30})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Read
	u, err := users.Read(ctx, "1")
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
	err = users.Update(ctx, u)
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	u2, _ := users.Read(ctx, "1")
	if u2.Name != "Alice Updated" {
		t.Errorf("expected 'Alice Updated', got %s", u2.Name)
	}
	if u2.Age != 31 {
		t.Errorf("expected age 31, got %d", u2.Age)
	}

	// Delete
	err = users.Delete(ctx, "1")
	if err != nil {
		t.Fatalf("delete: %v", err)
	}

	_, err = users.Read(ctx, "1")
	if err != model.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDuplicateKey(t *testing.T) {
	users := setup(t)
	ctx := context.Background()

	users.Create(ctx, &User{ID: "1", Name: "Alice"})
	err := users.Create(ctx, &User{ID: "1", Name: "Bob"})
	if err != model.ErrDuplicateKey {
		t.Errorf("expected ErrDuplicateKey, got %v", err)
	}
}

func TestNotFound(t *testing.T) {
	users := setup(t)
	ctx := context.Background()

	_, err := users.Read(ctx, "nonexistent")
	if err != model.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}

	err = users.Update(ctx, &User{ID: "nonexistent"})
	if err != model.ErrNotFound {
		t.Errorf("expected ErrNotFound on update, got %v", err)
	}

	err = users.Delete(ctx, "nonexistent")
	if err != model.ErrNotFound {
		t.Errorf("expected ErrNotFound on delete, got %v", err)
	}
}

func TestList(t *testing.T) {
	users := setup(t)
	ctx := context.Background()

	users.Create(ctx, &User{ID: "1", Name: "Alice", Age: 30})
	users.Create(ctx, &User{ID: "2", Name: "Bob", Age: 25})
	users.Create(ctx, &User{ID: "3", Name: "Charlie", Age: 35})

	// List all
	all, err := users.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("expected 3, got %d", len(all))
	}
}

func TestListWithFilter(t *testing.T) {
	users := setup(t)
	ctx := context.Background()

	users.Create(ctx, &User{ID: "1", Name: "Alice", Age: 30})
	users.Create(ctx, &User{ID: "2", Name: "Bob", Age: 25})
	users.Create(ctx, &User{ID: "3", Name: "Alice", Age: 35})

	// Filter by name
	results, err := users.List(ctx, model.Where("name", "Alice"))
	if err != nil {
		t.Fatalf("list with filter: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 Alices, got %d", len(results))
	}
}

func TestListWithLimitOffset(t *testing.T) {
	users := setup(t)
	ctx := context.Background()

	users.Create(ctx, &User{ID: "1", Name: "A", Age: 1})
	users.Create(ctx, &User{ID: "2", Name: "B", Age: 2})
	users.Create(ctx, &User{ID: "3", Name: "C", Age: 3})
	users.Create(ctx, &User{ID: "4", Name: "D", Age: 4})

	// Sort by name, get 2 starting from offset 1
	results, err := users.List(ctx,
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
	users := setup(t)
	ctx := context.Background()

	users.Create(ctx, &User{ID: "1", Name: "Alice", Age: 30})
	users.Create(ctx, &User{ID: "2", Name: "Bob", Age: 25})
	users.Create(ctx, &User{ID: "3", Name: "Alice", Age: 35})

	count, err := users.Count(ctx)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3, got %d", count)
	}

	count, err = users.Count(ctx, model.Where("name", "Alice"))
	if err != nil {
		t.Fatalf("count with filter: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}

func TestWhereOp(t *testing.T) {
	users := setup(t)
	ctx := context.Background()

	users.Create(ctx, &User{ID: "1", Name: "Alice", Age: 30})
	users.Create(ctx, &User{ID: "2", Name: "Bob", Age: 25})
	users.Create(ctx, &User{ID: "3", Name: "Charlie", Age: 35})

	// Age > 28
	results, err := users.List(ctx, model.WhereOp("age", ">", 28))
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 (age > 28), got %d", len(results))
	}
}
