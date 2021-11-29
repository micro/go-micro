package function

import (
	"go.m3o.com/client"
)

func NewFunctionService(token string) *FunctionService {
	return &FunctionService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type FunctionService struct {
	client *client.Client
}

// Call a function by name
func (t *FunctionService) Call(request *CallRequest) (*CallResponse, error) {
	rsp := &CallResponse{}
	return rsp, t.client.Call("function", "Call", request, rsp)
}

// Delete a function by name
func (t *FunctionService) Delete(request *DeleteRequest) (*DeleteResponse, error) {
	rsp := &DeleteResponse{}
	return rsp, t.client.Call("function", "Delete", request, rsp)
}

// Deploy a group of functions
func (t *FunctionService) Deploy(request *DeployRequest) (*DeployResponse, error) {
	rsp := &DeployResponse{}
	return rsp, t.client.Call("function", "Deploy", request, rsp)
}

// Get the info for a deployed function
func (t *FunctionService) Describe(request *DescribeRequest) (*DescribeResponse, error) {
	rsp := &DescribeResponse{}
	return rsp, t.client.Call("function", "Describe", request, rsp)
}

// List all the deployed functions
func (t *FunctionService) List(request *ListRequest) (*ListResponse, error) {
	rsp := &ListResponse{}
	return rsp, t.client.Call("function", "List", request, rsp)
}

type CallRequest struct {
	// Name of the function
	Name string `json:"name"`
	// Request body that will be passed to the function
	Request map[string]interface{} `json:"request"`
}

type CallResponse struct {
	// Response body that the function returned
	Response map[string]interface{} `json:"response"`
}

type DeleteRequest struct {
	// The name of the function
	Name string `json:"name"`
	// Optional project name
	Project string `json:"project"`
}

type DeleteResponse struct {
}

type DeployRequest struct {
	// entry point, ie. handler name in the source code
	// if not provided, defaults to the name parameter
	Entrypoint string `json:"entrypoint"`
	// environment variables to pass in at runtime
	EnvVars map[string]string `json:"envVars"`
	// function name
	Name string `json:"name"`
	// project is used for namespacing your functions
	// optional. defaults to "default".
	Project string `json:"project"`
	// github url to repo
	Repo string `json:"repo"`
	// runtime/language of the function
	// eg: php74,
	// nodejs6, nodejs8, nodejs10, nodejs12, nodejs14, nodejs16
	// dotnet3
	// java11
	// ruby26, ruby27
	// go111, go113, go116
	// python37, python38, python39
	Runtime string `json:"runtime"`
	// optional subfolder path
	Subfolder string `json:"subfolder"`
}

type DeployResponse struct {
}

type DescribeRequest struct {
	// The name of the function
	Name string `json:"name"`
	// Optional project name
	Project string `json:"project"`
}

type DescribeResponse struct {
	// The function requested
	Function *Func `json:"function"`
	// The timeout for requests to the function
	Timeout string `json:"timeout"`
	// The time at which the function was updated
	UpdatedAt string `json:"updatedAt"`
}

type Func struct {
	// name of handler in source code
	Entrypoint string `json:"entrypoint"`
	// function name
	// limitation: must be unique across projects
	Name string `json:"name"`
	// project of function, optional
	// defaults to literal "default"
	// used to namespace functions
	Project string `json:"project"`
	// git repo address
	Repo string `json:"repo"`
	// runtime/language of the function
	// eg: php74,
	// nodejs6, nodejs8, nodejs10, nodejs12, nodejs14, nodejs16
	// dotnet3
	// java11
	// ruby26, ruby27
	// go111, go113, go116
	// python37, python38, python39
	Runtime string `json:"runtime"`
	// eg. ACTIVE, DEPLOY_IN_PROGRESS, OFFLINE etc
	Status string `json:"status"`
	// subfolder path to entrypoint
	Subfolder string `json:"subfolder"`
}

type ListRequest struct {
	// optional project name
	Project string `json:"project"`
}

type ListResponse struct {
	// List of functions deployed
	Functions []Func `json:"functions"`
}
