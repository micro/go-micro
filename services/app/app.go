package app

import (
	"go-micro.dev/v4/api/client"
)

type App interface {
	Delete(*DeleteRequest) (*DeleteResponse, error)
	List(*ListRequest) (*ListResponse, error)
	Regions(*RegionsRequest) (*RegionsResponse, error)
	Reserve(*ReserveRequest) (*ReserveResponse, error)
	Resolve(*ResolveRequest) (*ResolveResponse, error)
	Run(*RunRequest) (*RunResponse, error)
	Status(*StatusRequest) (*StatusResponse, error)
	Update(*UpdateRequest) (*UpdateResponse, error)
}

func NewAppService(token string) *AppService {
	return &AppService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type AppService struct {
	client *client.Client
}

// Delete an app
func (t *AppService) Delete(request *DeleteRequest) (*DeleteResponse, error) {

	rsp := &DeleteResponse{}
	return rsp, t.client.Call("app", "Delete", request, rsp)

}

// List all the apps
func (t *AppService) List(request *ListRequest) (*ListResponse, error) {

	rsp := &ListResponse{}
	return rsp, t.client.Call("app", "List", request, rsp)

}

// Return the support regions
func (t *AppService) Regions(request *RegionsRequest) (*RegionsResponse, error) {

	rsp := &RegionsResponse{}
	return rsp, t.client.Call("app", "Regions", request, rsp)

}

// Reserve apps beyond the free quota. Call Run after.
func (t *AppService) Reserve(request *ReserveRequest) (*ReserveResponse, error) {

	rsp := &ReserveResponse{}
	return rsp, t.client.Call("app", "Reserve", request, rsp)

}

// Resolve an app by id to its raw backend endpoint
func (t *AppService) Resolve(request *ResolveRequest) (*ResolveResponse, error) {

	rsp := &ResolveResponse{}
	return rsp, t.client.Call("app", "Resolve", request, rsp)

}

// Run an app from a source repo. Specify region etc.
func (t *AppService) Run(request *RunRequest) (*RunResponse, error) {

	rsp := &RunResponse{}
	return rsp, t.client.Call("app", "Run", request, rsp)

}

// Get the status of an app
func (t *AppService) Status(request *StatusRequest) (*StatusResponse, error) {

	rsp := &StatusResponse{}
	return rsp, t.client.Call("app", "Status", request, rsp)

}

// Update the app. The latest source code will be downloaded, built and deployed.
func (t *AppService) Update(request *UpdateRequest) (*UpdateResponse, error) {

	rsp := &UpdateResponse{}
	return rsp, t.client.Call("app", "Update", request, rsp)

}

type DeleteRequest struct {
	// name of the app
	Name string `json:"name"`
}

type DeleteResponse struct {
}

type ListRequest struct {
}

type ListResponse struct {
	// all the apps
	Services []Service `json:"services"`
}

type RegionsRequest struct {
}

type RegionsResponse struct {
	Regions []string `json:"regions"`
}

type Reservation struct {
	// time of reservation
	Created string `json:"created"`
	// time reservation expires
	Expires string `json:"expires"`
	// name of the app
	Name string `json:"name"`
	// owner id
	Owner string `json:"owner"`
	// associated token
	Token string `json:"token"`
}

type ReserveRequest struct {
	// name of your app e.g helloworld
	Name string `json:"name"`
}

type ReserveResponse struct {
	// The app reservation
	Reservation *Reservation `json:"reservation"`
}

type ResolveRequest struct {
	// the service id
	Id string `json:"id"`
}

type ResolveResponse struct {
	// the end provider url
	Url string `json:"url"`
}

type RunRequest struct {
	// branch. defaults to master
	Branch string `json:"branch"`
	// associatede env vars to pass in
	EnvVars map[string]string `json:"env_vars"`
	// name of the app
	Name string `json:"name"`
	// port to run on
	Port int32 `json:"port"`
	// region to run in
	Region string `json:"region"`
	// source repository
	Repo string `json:"repo"`
}

type RunResponse struct {
	// The running service
	Service *Service `json:"service"`
}

type Service struct {
	// branch of code
	Branch string `json:"branch"`
	// time of creation
	Created string `json:"created"`
	// custom domains
	CustomDomains string `json:"custom_domains"`
	// associated env vars
	EnvVars map[string]string `json:"env_vars"`
	// unique id
	Id string `json:"id"`
	// name of the app
	Name string `json:"name"`
	// port running on
	Port int32 `json:"port"`
	// region running in
	Region string `json:"region"`
	// source repository
	Repo string `json:"repo"`
	// status of the app
	Status string `json:"status"`
	// last updated
	Updated string `json:"updated"`
	// app url
	Url string `json:"url"`
}

type StatusRequest struct {
	// name of the app
	Name string `json:"name"`
}

type StatusResponse struct {
	// running service info
	Service *Service `json:"service"`
}

type UpdateRequest struct {
	// name of the app
	Name string `json:"name"`
}

type UpdateResponse struct {
}
