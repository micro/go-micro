package db

import (
	"github.com/m3o/m3o-go/client"
)

func NewDbService(token string) *DbService {
	return &DbService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type DbService struct {
	client *client.Client
}

// Create a record in the database. Optionally include an "id" field otherwise it's set automatically.
func (t *DbService) Create(request *CreateRequest) (*CreateResponse, error) {
	rsp := &CreateResponse{}
	return rsp, t.client.Call("db", "Create", request, rsp)
}

// Delete a record in the database by id.
func (t *DbService) Delete(request *DeleteRequest) (*DeleteResponse, error) {
	rsp := &DeleteResponse{}
	return rsp, t.client.Call("db", "Delete", request, rsp)
}

// Read data from a table. Lookup can be by ID or via querying any field in the record.
func (t *DbService) Read(request *ReadRequest) (*ReadResponse, error) {
	rsp := &ReadResponse{}
	return rsp, t.client.Call("db", "Read", request, rsp)
}

// Truncate the records in a table
func (t *DbService) Truncate(request *TruncateRequest) (*TruncateResponse, error) {
	rsp := &TruncateResponse{}
	return rsp, t.client.Call("db", "Truncate", request, rsp)
}

// Update a record in the database. Include an "id" in the record to update.
func (t *DbService) Update(request *UpdateRequest) (*UpdateResponse, error) {
	rsp := &UpdateResponse{}
	return rsp, t.client.Call("db", "Update", request, rsp)
}

type CreateRequest struct {
	// JSON encoded record or records (can be array or object)
	Record map[string]interface{} `json:"record"`
	// Optional table name. Defaults to 'default'
	Table string `json:"table"`
}

type CreateResponse struct {
	// The id of the record (either specified or automatically created)
	Id string `json:"id"`
}

type DeleteRequest struct {
	// id of the record
	Id string `json:"id"`
	// Optional table name. Defaults to 'default'
	Table string `json:"table"`
}

type DeleteResponse struct {
}

type ReadRequest struct {
	// Read by id. Equivalent to 'id == "your-id"'
	Id string `json:"id"`
	// Maximum number of records to return. Default limit is 25.
	// Maximum limit is 1000. Anything higher will return an error.
	Limit  int32 `json:"limit"`
	Offset int32 `json:"offset"`
	// 'asc' (default), 'desc'
	Order string `json:"order"`
	// field name to order by
	OrderBy string `json:"orderBy"`
	// Examples: 'age >= 18', 'age >= 18 and verified == true'
	// Comparison operators: '==', '!=', '<', '>', '<=', '>='
	// Logical operator: 'and'
	// Dot access is supported, eg: 'user.age == 11'
	// Accessing list elements is not supported yet.
	Query string `json:"query"`
	// Optional table name. Defaults to 'default'
	Table string `json:"table"`
}

type ReadResponse struct {
	// JSON encoded records
	Records []map[string]interface{} `json:"records"`
}

type TruncateRequest struct {
	// Optional table name. Defaults to 'default'
	Table string `json:"table"`
}

type TruncateResponse struct {
	// The table truncated
	Table string `json:"table"`
}

type UpdateRequest struct {
	// The id of the record. If not specified it is inferred from the 'id' field of the record
	Id string `json:"id"`
	// record, JSON object
	Record map[string]interface{} `json:"record"`
	// Optional table name. Defaults to 'default'
	Table string `json:"table"`
}

type UpdateResponse struct {
}
