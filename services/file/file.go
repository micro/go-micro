package file

import (
	"go-micro.dev/v4/api/client"
)

type File interface {
	Delete(*DeleteRequest) (*DeleteResponse, error)
	List(*ListRequest) (*ListResponse, error)
	Read(*ReadRequest) (*ReadResponse, error)
	Save(*SaveRequest) (*SaveResponse, error)
}

func NewFileService(token string) *FileService {
	return &FileService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type FileService struct {
	client *client.Client
}

// Delete a file by project name/path
func (t *FileService) Delete(request *DeleteRequest) (*DeleteResponse, error) {

	rsp := &DeleteResponse{}
	return rsp, t.client.Call("file", "Delete", request, rsp)

}

// List files by their project and optionally a path.
func (t *FileService) List(request *ListRequest) (*ListResponse, error) {

	rsp := &ListResponse{}
	return rsp, t.client.Call("file", "List", request, rsp)

}

// Read a file by path
func (t *FileService) Read(request *ReadRequest) (*ReadResponse, error) {

	rsp := &ReadResponse{}
	return rsp, t.client.Call("file", "Read", request, rsp)

}

// Save a file
func (t *FileService) Save(request *SaveRequest) (*SaveResponse, error) {

	rsp := &SaveResponse{}
	return rsp, t.client.Call("file", "Save", request, rsp)

}

type DeleteRequest struct {
	// Path to the file
	Path string `json:"path"`
	// The project name
	Project string `json:"project"`
}

type DeleteResponse struct {
}

type ListRequest struct {
	// Defaults to '/', ie. lists all files in a project.
	// Supply path to a folder if you want to list
	// files inside that folder
	// eg. '/docs'
	Path string `json:"path"`
	// Project, required for listing.
	Project string `json:"project"`
}

type ListResponse struct {
	Files []Record `json:"files"`
}

type ReadRequest struct {
	// Path to the file
	Path string `json:"path"`
	// Project name
	Project string `json:"project"`
}

type ReadResponse struct {
	// Returns the file
	File *Record `json:"file"`
}

type Record struct {
	// File contents
	Content string `json:"content"`
	// Time the file was created e.g 2021-05-20T13:37:21Z
	Created string `json:"created"`
	// Any other associated metadata as a map of key-value pairs
	Metadata map[string]string `json:"metadata"`
	// Path to file or folder eg. '/documents/text-files/file.txt'.
	Path string `json:"path"`
	// A custom project to group files
	// eg. file-of-mywebsite.com
	Project string `json:"project"`
	// Time the file was updated e.g 2021-05-20T13:37:21Z
	Updated string `json:"updated"`
}

type SaveRequest struct {
	File *Record `json:"file"`
}

type SaveResponse struct {
}
