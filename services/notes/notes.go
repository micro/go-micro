package notes

import (
	"github.com/m3o/m3o-go/client"
)

func NewNotesService(token string) *NotesService {
	return &NotesService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type NotesService struct {
	client *client.Client
}

// Create a new note
func (t *NotesService) Create(request *CreateRequest) (*CreateResponse, error) {
	rsp := &CreateResponse{}
	return rsp, t.client.Call("notes", "Create", request, rsp)
}

// Delete a note
func (t *NotesService) Delete(request *DeleteRequest) (*DeleteResponse, error) {
	rsp := &DeleteResponse{}
	return rsp, t.client.Call("notes", "Delete", request, rsp)
}

// List all the notes
func (t *NotesService) List(request *ListRequest) (*ListResponse, error) {
	rsp := &ListResponse{}
	return rsp, t.client.Call("notes", "List", request, rsp)
}

// Read a note
func (t *NotesService) Read(request *ReadRequest) (*ReadResponse, error) {
	rsp := &ReadResponse{}
	return rsp, t.client.Call("notes", "Read", request, rsp)
}

// Specify the note to events
func (t *NotesService) Subscribe(request *SubscribeRequest) (*SubscribeResponse, error) {
	rsp := &SubscribeResponse{}
	return rsp, t.client.Call("notes", "Subscribe", request, rsp)
}

// Update a note
func (t *NotesService) Update(request *UpdateRequest) (*UpdateResponse, error) {
	rsp := &UpdateResponse{}
	return rsp, t.client.Call("notes", "Update", request, rsp)
}

type CreateRequest struct {
	// note text
	Text string `json:"text"`
	// note title
	Title string `json:"title"`
}

type CreateResponse struct {
	// The created note
	Note *Note `json:"note"`
}

type DeleteRequest struct {
	// specify the id of the note
	Id string `json:"id"`
}

type DeleteResponse struct {
	Note *Note `json:"note"`
}

type ListRequest struct {
}

type ListResponse struct {
	// the list of notes
	Notes []Note `json:"notes"`
}

type Note struct {
	// time at which the note was created
	Created string `json:"created"`
	// unique id for the note, generated if not specified
	Id string `json:"id"`
	// text within the note
	Text string `json:"text"`
	// title of the note
	Title string `json:"title"`
	// time at which the note was updated
	Updated string `json:"updated"`
}

type ReadRequest struct {
	// the note id
	Id string `json:"id"`
}

type ReadResponse struct {
	// The note
	Note *Note `json:"note"`
}

type SubscribeRequest struct {
	// optionally specify a note id
	Id string `json:"id"`
}

type SubscribeResponse struct {
	// the event which occured; created, deleted, updated
	Event string `json:"event"`
	// the note which the operation occured on
	Note *Note `json:"note"`
}

type UpdateRequest struct {
	Note *Note `json:"note"`
}

type UpdateResponse struct {
	Note *Note `json:"note"`
}
