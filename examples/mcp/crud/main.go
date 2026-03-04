// CRUD example: a contact book service with full MCP integration.
//
// This shows a realistic service with create, read, update, delete, and
// search operations, all automatically exposed as MCP tools with rich
// documentation for AI agents.
//
// Run:
//
//	go run .
//
// MCP tools: http://localhost:3001/mcp/tools
// Test:      curl http://localhost:3001/mcp/tools | jq
package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"go-micro.dev/v5"
	"go-micro.dev/v5/gateway/mcp"
)

// --- Types ---

// Contact represents a person in the contact book.
type Contact struct {
	ID    string `json:"id" description:"Unique contact identifier"`
	Name  string `json:"name" description:"Full name"`
	Email string `json:"email" description:"Email address"`
	Phone string `json:"phone" description:"Phone number in E.164 format"`
	Role  string `json:"role" description:"Job title or role"`
	Notes string `json:"notes" description:"Free-text notes about this contact"`
}

type CreateRequest struct {
	Name  string `json:"name" description:"Full name (required)"`
	Email string `json:"email" description:"Email address (required)"`
	Phone string `json:"phone" description:"Phone number"`
	Role  string `json:"role" description:"Job title or role"`
	Notes string `json:"notes" description:"Free-text notes"`
}

type CreateResponse struct {
	Contact *Contact `json:"contact" description:"The newly created contact"`
}

type GetRequest struct {
	ID string `json:"id" description:"Contact ID to look up"`
}

type GetResponse struct {
	Contact *Contact `json:"contact" description:"The requested contact"`
}

type UpdateRequest struct {
	ID    string `json:"id" description:"Contact ID to update (required)"`
	Name  string `json:"name" description:"New name (leave empty to keep current)"`
	Email string `json:"email" description:"New email (leave empty to keep current)"`
	Phone string `json:"phone" description:"New phone (leave empty to keep current)"`
	Role  string `json:"role" description:"New role (leave empty to keep current)"`
	Notes string `json:"notes" description:"New notes (leave empty to keep current)"`
}

type UpdateResponse struct {
	Contact *Contact `json:"contact" description:"The updated contact"`
}

type DeleteRequest struct {
	ID string `json:"id" description:"Contact ID to delete"`
}

type DeleteResponse struct {
	Deleted bool `json:"deleted" description:"True if the contact was deleted"`
}

type ListRequest struct {
}

type ListResponse struct {
	Contacts []*Contact `json:"contacts" description:"All contacts in the book"`
}

type SearchRequest struct {
	Query string `json:"query" description:"Search term to match against name, email, role, or notes"`
}

type SearchResponse struct {
	Contacts []*Contact `json:"contacts" description:"Contacts matching the search query"`
}

// --- Handler ---

// Contacts manages a contact book with CRUD operations.
type Contacts struct {
	mu      sync.RWMutex
	store   map[string]*Contact
	counter int
}

func NewContacts() *Contacts {
	c := &Contacts{store: make(map[string]*Contact)}
	// Seed with example data
	c.store["c-1"] = &Contact{ID: "c-1", Name: "Alice Johnson", Email: "alice@example.com", Phone: "+1-555-0101", Role: "Engineer", Notes: "Backend team lead"}
	c.store["c-2"] = &Contact{ID: "c-2", Name: "Bob Smith", Email: "bob@example.com", Phone: "+1-555-0102", Role: "Designer", Notes: "UI/UX specialist"}
	c.store["c-3"] = &Contact{ID: "c-3", Name: "Carol Davis", Email: "carol@example.com", Phone: "+1-555-0103", Role: "PM", Notes: "Leads the platform team"}
	c.counter = 3
	return c
}

// Create adds a new contact to the book. Name and email are required.
//
// @example {"name": "Dave Wilson", "email": "dave@example.com", "role": "Engineer"}
func (h *Contacts) Create(ctx context.Context, req *CreateRequest, rsp *CreateResponse) error {
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}
	if req.Email == "" {
		return fmt.Errorf("email is required")
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	h.counter++
	id := fmt.Sprintf("c-%d", h.counter)
	contact := &Contact{
		ID:    id,
		Name:  req.Name,
		Email: req.Email,
		Phone: req.Phone,
		Role:  req.Role,
		Notes: req.Notes,
	}
	h.store[id] = contact
	rsp.Contact = contact
	return nil
}

// Get retrieves a single contact by ID.
//
// @example {"id": "c-1"}
func (h *Contacts) Get(ctx context.Context, req *GetRequest, rsp *GetResponse) error {
	if req.ID == "" {
		return fmt.Errorf("id is required")
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	contact, ok := h.store[req.ID]
	if !ok {
		return fmt.Errorf("contact %s not found", req.ID)
	}
	rsp.Contact = contact
	return nil
}

// Update modifies an existing contact. Only non-empty fields are updated,
// so you can change just the email without affecting other fields.
//
// @example {"id": "c-1", "role": "Senior Engineer"}
func (h *Contacts) Update(ctx context.Context, req *UpdateRequest, rsp *UpdateResponse) error {
	if req.ID == "" {
		return fmt.Errorf("id is required")
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	contact, ok := h.store[req.ID]
	if !ok {
		return fmt.Errorf("contact %s not found", req.ID)
	}

	if req.Name != "" {
		contact.Name = req.Name
	}
	if req.Email != "" {
		contact.Email = req.Email
	}
	if req.Phone != "" {
		contact.Phone = req.Phone
	}
	if req.Role != "" {
		contact.Role = req.Role
	}
	if req.Notes != "" {
		contact.Notes = req.Notes
	}

	rsp.Contact = contact
	return nil
}

// Delete removes a contact from the book permanently.
//
// @example {"id": "c-1"}
func (h *Contacts) Delete(ctx context.Context, req *DeleteRequest, rsp *DeleteResponse) error {
	if req.ID == "" {
		return fmt.Errorf("id is required")
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.store[req.ID]; !ok {
		return fmt.Errorf("contact %s not found", req.ID)
	}

	delete(h.store, req.ID)
	rsp.Deleted = true
	return nil
}

// List returns all contacts in the book.
//
// @example {}
func (h *Contacts) List(ctx context.Context, req *ListRequest, rsp *ListResponse) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, c := range h.store {
		rsp.Contacts = append(rsp.Contacts, c)
	}
	return nil
}

// Search finds contacts matching a query string. Matches against name,
// email, role, and notes fields (case-insensitive).
//
// @example {"query": "engineer"}
func (h *Contacts) Search(ctx context.Context, req *SearchRequest, rsp *SearchResponse) error {
	if req.Query == "" {
		return fmt.Errorf("query is required")
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	q := strings.ToLower(req.Query)
	for _, c := range h.store {
		if strings.Contains(strings.ToLower(c.Name), q) ||
			strings.Contains(strings.ToLower(c.Email), q) ||
			strings.Contains(strings.ToLower(c.Role), q) ||
			strings.Contains(strings.ToLower(c.Notes), q) {
			rsp.Contacts = append(rsp.Contacts, c)
		}
	}
	return nil
}

func main() {
	service := micro.New("contacts",
		micro.Address(":9010"),
		mcp.WithMCP(":3001"),
	)
	service.Init()

	if err := service.Handle(NewContacts()); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Contacts service running on :9010")
	fmt.Println("MCP tools available at http://localhost:3001/mcp/tools")
	fmt.Println()
	fmt.Println("Try asking an AI agent:")
	fmt.Println("  'List all contacts'")
	fmt.Println("  'Find engineers in the contact book'")
	fmt.Println("  'Add a new contact for Eve at eve@example.com'")
	fmt.Println("  'Update Alice's role to Staff Engineer'")

	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}
