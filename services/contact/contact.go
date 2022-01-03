package contact

import (
	"go-micro.dev/v4/api/client"
)

type Contact interface {
	Create(*CreateRequest) (*CreateResponse, error)
	Delete(*DeleteRequest) (*DeleteResponse, error)
	List(*ListRequest) (*ListResponse, error)
	Read(*ReadRequest) (*ReadResponse, error)
	Update(*UpdateRequest) (*UpdateResponse, error)
}

func NewContactService(token string) *ContactService {
	return &ContactService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type ContactService struct {
	client *client.Client
}

//
func (t *ContactService) Create(request *CreateRequest) (*CreateResponse, error) {

	rsp := &CreateResponse{}
	return rsp, t.client.Call("contact", "Create", request, rsp)

}

//
func (t *ContactService) Delete(request *DeleteRequest) (*DeleteResponse, error) {

	rsp := &DeleteResponse{}
	return rsp, t.client.Call("contact", "Delete", request, rsp)

}

//
func (t *ContactService) List(request *ListRequest) (*ListResponse, error) {

	rsp := &ListResponse{}
	return rsp, t.client.Call("contact", "List", request, rsp)

}

//
func (t *ContactService) Read(request *ReadRequest) (*ReadResponse, error) {

	rsp := &ReadResponse{}
	return rsp, t.client.Call("contact", "Read", request, rsp)

}

//
func (t *ContactService) Update(request *UpdateRequest) (*UpdateResponse, error) {

	rsp := &UpdateResponse{}
	return rsp, t.client.Call("contact", "Update", request, rsp)

}

type Address struct {
	// the label of the address
	Label string `json:"label"`
	// the address location
	Location string `json:"location"`
}

type ContactInfo struct {
	// the address
	Addresses []Address `json:"addresses"`
	// the birthday
	Birthday string `json:"birthday"`
	// create date string in RFC3339
	CreatedAt string `json:"created_at"`
	// the emails
	Emails []Email `json:"emails"`
	// contact id
	Id string `json:"id"`
	// the contact links
	Links []Link `json:"links"`
	// the contact name
	Name string `json:"name"`
	// note of the contact
	Note string `json:"note"`
	// the phone numbers
	Phones []Phone `json:"phones"`
	// the social media username
	SocialMedias *SocialMedia `json:"social_medias"`
	// update date string in RFC3339
	UpdatedAt string `json:"updated_at"`
}

type CreateRequest struct {
	// optional, location
	Addresses []Address `json:"addresses"`
	// optional, birthday
	Birthday string `json:"birthday"`
	// optional, emails
	Emails []Email `json:"emails"`
	// optional, links
	Links []Link `json:"links"`
	// required, the name of the contact
	Name string `json:"name"`
	// optional, note of the contact
	Note string `json:"note"`
	// optional, phone numbers
	Phones []Phone `json:"phones"`
	// optional, social media
	SocialMedias *SocialMedia `json:"social_medias"`
}

type CreateResponse struct {
	Contact *ContactInfo `json:"contact"`
}

type DeleteRequest struct {
	// the id of the contact
	Id string `json:"id"`
}

type DeleteResponse struct {
}

type Email struct {
	// the email address
	Address string `json:"address"`
	// the label of the email
	Label string `json:"label"`
}

type Link struct {
	// the label of the link
	Label string `json:"label"`
	// the url of the contact
	Url string `json:"url"`
}

type ListRequest struct {
	// optional, default is 30
	Limit int32 `json:"limit"`
	// optional
	Offset int32 `json:"offset"`
}

type ListResponse struct {
	Contacts []ContactInfo `json:"contacts"`
}

type Phone struct {
	// the label of the phone number
	Label string `json:"label"`
	// phone number
	Number string `json:"number"`
}

type ReadRequest struct {
	Id string `json:"id"`
}

type ReadResponse struct {
	Contact *ContactInfo `json:"contact"`
}

type SocialMedia struct {
	// the label of the social
	Label string `json:"label"`
	// the username of social media
	Username string `json:"username"`
}

type UpdateRequest struct {
	// optional, addresses
	Addresses []Address `json:"addresses"`
	// optional, birthday
	Birthday string `json:"birthday"`
	// optional, emails
	Emails []Email `json:"emails"`
	// required, the contact id
	Id string `json:"id"`
	// optional, links
	Links []Link `json:"links"`
	// required, the name
	Name string `json:"name"`
	// optional, note
	Note string `json:"note"`
	// optional, phone number
	Phones []Phone `json:"phones"`
	// optional, social media
	SocialMedias *SocialMedia `json:"social_medias"`
}

type UpdateResponse struct {
	Contact *ContactInfo `json:"contact"`
}
