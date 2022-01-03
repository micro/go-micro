package db

import (
	"go-micro.dev/v4/api/client"
)

type Db interface {
	Count(*CountRequest) (*CountResponse, error)
	Create(*CreateRequest) (*CreateResponse, error)
	Delete(*DeleteRequest) (*DeleteResponse, error)
	DropTable(*DropTableRequest) (*DropTableResponse, error)
	ListTables(*ListTablesRequest) (*ListTablesResponse, error)
	Read(*ReadRequest) (*ReadResponse, error)
	RenameTable(*RenameTableRequest) (*RenameTableResponse, error)
	Truncate(*TruncateRequest) (*TruncateResponse, error)
	Update(*UpdateRequest) (*UpdateResponse, error)
}

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

// Count records in a table
func (t *DbService) Count(request *CountRequest) (*CountResponse, error) {

	rsp := &CountResponse{}
	return rsp, t.client.Call("db", "Count", request, rsp)

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

// Drop a table in the DB
func (t *DbService) DropTable(request *DropTableRequest) (*DropTableResponse, error) {

	rsp := &DropTableResponse{}
	return rsp, t.client.Call("db", "DropTable", request, rsp)

}

// List tables in the DB
func (t *DbService) ListTables(request *ListTablesRequest) (*ListTablesResponse, error) {

	rsp := &ListTablesResponse{}
	return rsp, t.client.Call("db", "ListTables", request, rsp)

}

// Read data from a table. Lookup can be by ID or via querying any field in the record.
func (t *DbService) Read(request *ReadRequest) (*ReadResponse, error) {

	rsp := &ReadResponse{}
	return rsp, t.client.Call("db", "Read", request, rsp)

}

// Rename a table
func (t *DbService) RenameTable(request *RenameTableRequest) (*RenameTableResponse, error) {

	rsp := &RenameTableResponse{}
	return rsp, t.client.Call("db", "RenameTable", request, rsp)

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

type CountRequest struct {
	// specify the table name
	Table string `json:"table"`
}

type CountResponse struct {
	// the number of records in the table
	Count int32 `json:"count"`
}

type CreateRequest struct {
	// optional record id to use
	Id string `json:"id"`
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

type DropTableRequest struct {
	Table string `json:"table"`
}

type DropTableResponse struct {
}

type ListTablesRequest struct {
}

type ListTablesResponse struct {
	// list of tables
	Tables []string `json:"tables"`
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

type RenameTableRequest struct {
	// current table name
	From string `json:"from"`
	// new table name
	To string `json:"to"`
}

type RenameTableResponse struct {
}

type TruncateRequest struct {
	Table string `json:"table"`
}

type TruncateResponse struct {
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
