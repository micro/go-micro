package search

import (
	"go-micro.dev/v4/api/client"
)

type Search interface {
	DeleteIndex(*DeleteIndexRequest) (*DeleteIndexResponse, error)
	Delete(*DeleteRequest) (*DeleteResponse, error)
	Index(*IndexRequest) (*IndexResponse, error)
	Search(*SearchRequest) (*SearchResponse, error)
}

func NewSearchService(token string) *SearchService {
	return &SearchService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type SearchService struct {
	client *client.Client
}

// Delete an index.
func (t *SearchService) DeleteIndex(request *DeleteIndexRequest) (*DeleteIndexResponse, error) {

	rsp := &DeleteIndexResponse{}
	return rsp, t.client.Call("search", "DeleteIndex", request, rsp)

}

// Delete a document given its ID
func (t *SearchService) Delete(request *DeleteRequest) (*DeleteResponse, error) {

	rsp := &DeleteResponse{}
	return rsp, t.client.Call("search", "Delete", request, rsp)

}

// Index a document i.e. insert a document to search for.
func (t *SearchService) Index(request *IndexRequest) (*IndexResponse, error) {

	rsp := &IndexResponse{}
	return rsp, t.client.Call("search", "Index", request, rsp)

}

// Search for documents in a given in index
func (t *SearchService) Search(request *SearchRequest) (*SearchResponse, error) {

	rsp := &SearchResponse{}
	return rsp, t.client.Call("search", "Search", request, rsp)

}

type CreateIndexRequest struct {
	Fields []Field `json:"fields"`
	// the name of the index
	Index string `json:"index"`
}

type CreateIndexResponse struct {
}

type DeleteIndexRequest struct {
	// The name of the index to delete
	Index string `json:"index"`
}

type DeleteIndexResponse struct {
}

type DeleteRequest struct {
	// The ID of the document to delete
	Id string `json:"id"`
	// The index the document belongs to
	Index string `json:"index"`
}

type DeleteResponse struct {
}

type Document struct {
	// The JSON contents of the document
	Contents map[string]interface{} `json:"contents"`
	// The ID for this document. If blank, one will be generated
	Id string `json:"id"`
}

type Field struct {
	// The name of the field. Use a `.` separator to define nested fields e.g. foo.bar
	Name string `json:"name"`
	// The type of the field - string, number
	Type string `json:"type"`
}

type IndexRequest struct {
	// The document to index
	Document *Document `json:"document"`
	// The index this document belongs to
	Index string `json:"index"`
}

type IndexResponse struct {
	Id string `json:"id"`
}

type SearchRequest struct {
	// The index the document belongs to
	Index string `json:"index"`
	// The query. See docs for query language examples
	Query string `json:"query"`
}

type SearchResponse struct {
	// The matching documents
	Documents []Document `json:"documents"`
}
