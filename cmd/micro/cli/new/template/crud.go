package template

var (
	CrudProtoSRV = `syntax = "proto3";

package {{dehyphen .Alias}};

option go_package = "./proto;{{dehyphen .Alias}}";

service {{title .Alias}} {
	rpc Create(CreateRequest) returns (CreateResponse) {}
	rpc Read(ReadRequest) returns (ReadResponse) {}
	rpc Update(UpdateRequest) returns (UpdateResponse) {}
	rpc Delete(DeleteRequest) returns (DeleteResponse) {}
	rpc List(ListRequest) returns (ListResponse) {}
}

message {{title .Alias}}Record {
	string id = 1;
	string name = 2;
	string email = 3;
	string phone = 4;
	string company = 5;
	int64 created = 6;
	int64 updated = 7;
}

message CreateRequest {
	string name = 1;
	string email = 2;
	string phone = 3;
	string company = 4;
}

message CreateResponse {
	{{title .Alias}}Record record = 1;
}

message ReadRequest {
	string id = 1;
}

message ReadResponse {
	{{title .Alias}}Record record = 1;
}

message UpdateRequest {
	string id = 1;
	string name = 2;
	string email = 3;
	string phone = 4;
	string company = 5;
}

message UpdateResponse {
	{{title .Alias}}Record record = 1;
}

message DeleteRequest {
	string id = 1;
}

message DeleteResponse {
	bool deleted = 1;
}

message ListRequest {
	int64 limit = 1;
	int64 offset = 2;
}

message ListResponse {
	repeated {{title .Alias}}Record records = 1;
	int64 total = 2;
}
`

	CrudHandlerSRV = `package handler

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	log "go-micro.dev/v6/logger"

	pb "{{.Dir}}/proto"
)

type {{title .Alias}} struct {
	mu      sync.RWMutex
	records map[string]*pb.{{title .Alias}}Record
}

func New() *{{title .Alias}} {
	return &{{title .Alias}}{
		records: make(map[string]*pb.{{title .Alias}}Record),
	}
}

// Create adds a new record and returns it with a generated ID.
//
// @example {"name": "Alice Smith", "email": "alice@example.com", "phone": "+1-555-0100", "company": "Acme Inc"}
func (h *{{title .Alias}}) Create(ctx context.Context, req *pb.CreateRequest, rsp *pb.CreateResponse) error {
	log.Infof("Creating record: %s", req.Name)

	now := time.Now().Unix()
	record := &pb.{{title .Alias}}Record{
		Id:      uuid.New().String(),
		Name:    req.Name,
		Email:   req.Email,
		Phone:   req.Phone,
		Company: req.Company,
		Created: now,
		Updated: now,
	}

	h.mu.Lock()
	h.records[record.Id] = record
	h.mu.Unlock()

	rsp.Record = record
	return nil
}

// Read retrieves a record by ID.
//
// @example {"id": "some-uuid"}
func (h *{{title .Alias}}) Read(ctx context.Context, req *pb.ReadRequest, rsp *pb.ReadResponse) error {
	h.mu.RLock()
	record, ok := h.records[req.Id]
	h.mu.RUnlock()

	if !ok {
		return fmt.Errorf("record %s not found", req.Id)
	}

	rsp.Record = record
	return nil
}

// Update modifies an existing record. Only non-empty fields are updated.
//
// @example {"id": "some-uuid", "name": "Alice Johnson", "email": "alice.j@example.com"}
func (h *{{title .Alias}}) Update(ctx context.Context, req *pb.UpdateRequest, rsp *pb.UpdateResponse) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	record, ok := h.records[req.Id]
	if !ok {
		return fmt.Errorf("record %s not found", req.Id)
	}

	if req.Name != "" {
		record.Name = req.Name
	}
	if req.Email != "" {
		record.Email = req.Email
	}
	if req.Phone != "" {
		record.Phone = req.Phone
	}
	if req.Company != "" {
		record.Company = req.Company
	}
	record.Updated = time.Now().Unix()

	rsp.Record = record
	return nil
}

// Delete removes a record by ID.
//
// @example {"id": "some-uuid"}
func (h *{{title .Alias}}) Delete(ctx context.Context, req *pb.DeleteRequest, rsp *pb.DeleteResponse) error {
	h.mu.Lock()
	_, ok := h.records[req.Id]
	if ok {
		delete(h.records, req.Id)
	}
	h.mu.Unlock()

	rsp.Deleted = ok
	return nil
}

// List returns all records with optional pagination.
//
// @example {"limit": 10, "offset": 0}
func (h *{{title .Alias}}) List(ctx context.Context, req *pb.ListRequest, rsp *pb.ListResponse) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	all := make([]*pb.{{title .Alias}}Record, 0, len(h.records))
	for _, r := range h.records {
		all = append(all, r)
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].Created > all[j].Created
	})

	rsp.Total = int64(len(all))

	offset := int(req.Offset)
	if offset > len(all) {
		offset = len(all)
	}
	limit := int(req.Limit)
	if limit <= 0 {
		limit = 20
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}

	rsp.Records = all[offset:end]
	return nil
}
`
)
