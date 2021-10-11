package user

import (
	"github.com/m3o/m3o-go/client"
)

func NewUserService(token string) *UserService {
	return &UserService{
		client: client.NewClient(&client.Options{
			Token: token,
		}),
	}
}

type UserService struct {
	client *client.Client
}

// Create a new user account. The email address and username for the account must be unique.
func (t *UserService) Create(request *CreateRequest) (*CreateResponse, error) {
	rsp := &CreateResponse{}
	return rsp, t.client.Call("user", "Create", request, rsp)
}

// Delete an account by id
func (t *UserService) Delete(request *DeleteRequest) (*DeleteResponse, error) {
	rsp := &DeleteResponse{}
	return rsp, t.client.Call("user", "Delete", request, rsp)
}

// Login using username or email. The response will return a new session for successful login,
// 401 in the case of login failure and 500 for any other error
func (t *UserService) Login(request *LoginRequest) (*LoginResponse, error) {
	rsp := &LoginResponse{}
	return rsp, t.client.Call("user", "Login", request, rsp)
}

// Logout a user account
func (t *UserService) Logout(request *LogoutRequest) (*LogoutResponse, error) {
	rsp := &LogoutResponse{}
	return rsp, t.client.Call("user", "Logout", request, rsp)
}

// Read an account by id, username or email. Only one need to be specified.
func (t *UserService) Read(request *ReadRequest) (*ReadResponse, error) {
	rsp := &ReadResponse{}
	return rsp, t.client.Call("user", "Read", request, rsp)
}

// Read a session by the session id. In the event it has expired or is not found and error is returned.
func (t *UserService) ReadSession(request *ReadSessionRequest) (*ReadSessionResponse, error) {
	rsp := &ReadSessionResponse{}
	return rsp, t.client.Call("user", "ReadSession", request, rsp)
}

// Send a verification email
// to the user being signed up. Email from will be from 'support@m3o.com',
// but you can provide the title and contents.
// The verification link will be injected in to the email as a template variable, $micro_verification_link.
// Example: 'Hi there, welcome onboard! Use the link below to verify your email: $micro_verification_link'
// The variable will be replaced with an actual url that will look similar to this:
// 'https://user.m3o.com/user/verify?token=a-verification-token&redirectUrl=your-redir-url'
func (t *UserService) SendVerificationEmail(request *SendVerificationEmailRequest) (*SendVerificationEmailResponse, error) {
	rsp := &SendVerificationEmailResponse{}
	return rsp, t.client.Call("user", "SendVerificationEmail", request, rsp)
}

// Update the account password
func (t *UserService) UpdatePassword(request *UpdatePasswordRequest) (*UpdatePasswordResponse, error) {
	rsp := &UpdatePasswordResponse{}
	return rsp, t.client.Call("user", "UpdatePassword", request, rsp)
}

// Update the account username or email
func (t *UserService) Update(request *UpdateRequest) (*UpdateResponse, error) {
	rsp := &UpdateResponse{}
	return rsp, t.client.Call("user", "Update", request, rsp)
}

// Verify the email address of an account from a token sent in an email to the user.
func (t *UserService) VerifyEmail(request *VerifyEmailRequest) (*VerifyEmailResponse, error) {
	rsp := &VerifyEmailResponse{}
	return rsp, t.client.Call("user", "VerifyEmail", request, rsp)
}

type Account struct {
	// unix timestamp
	Created int64 `json:"created,string"`
	// an email address
	Email string `json:"email"`
	// unique account id
	Id string `json:"id"`
	// Store any custom data you want about your users in this fields.
	Profile map[string]string `json:"profile"`
	// unix timestamp
	Updated int64 `json:"updated,string"`
	// alphanumeric username
	Username         string `json:"username"`
	VerificationDate int64  `json:"verificationDate,string"`
	Verified         bool   `json:"verified"`
}

type CreateRequest struct {
	// the email address
	Email string `json:"email"`
	// optional account id
	Id string `json:"id"`
	// the user password
	Password string `json:"password"`
	// optional user profile as map<string,string>
	Profile map[string]string `json:"profile"`
	// the username
	Username string `json:"username"`
}

type CreateResponse struct {
	Account *Account `json:"account"`
}

type DeleteRequest struct {
	// the account id
	Id string `json:"id"`
}

type DeleteResponse struct {
}

type LoginRequest struct {
	// The email address of the user
	Email string `json:"email"`
	// The password of the user
	Password string `json:"password"`
	// The username of the user
	Username string `json:"username"`
}

type LoginResponse struct {
	// The session of the logged in  user
	Session *Session `json:"session"`
}

type LogoutRequest struct {
	SessionId string `json:"sessionId"`
}

type LogoutResponse struct {
}

type ReadRequest struct {
	// the account email
	Email string `json:"email"`
	// the account id
	Id string `json:"id"`
	// the account username
	Username string `json:"username"`
}

type ReadResponse struct {
	Account *Account `json:"account"`
}

type ReadSessionRequest struct {
	// The unique session id
	SessionId string `json:"sessionId"`
}

type ReadSessionResponse struct {
	Session *Session `json:"session"`
}

type SendVerificationEmailRequest struct {
	Email              string `json:"email"`
	FailureRedirectUrl string `json:"failureRedirectUrl"`
	// Display name of the sender for the email. Note: the email address will still be 'support@m3o.com'
	FromName    string `json:"fromName"`
	RedirectUrl string `json:"redirectUrl"`
	Subject     string `json:"subject"`
	// Text content of the email. Don't forget to include the string '$micro_verification_link' which will be replaced by the real verification link
	// HTML emails are not available currently.
	TextContent string `json:"textContent"`
}

type SendVerificationEmailResponse struct {
}

type Session struct {
	// unix timestamp
	Created int64 `json:"created,string"`
	// unix timestamp
	Expires int64 `json:"expires,string"`
	// the session id
	Id string `json:"id"`
	// the associated user id
	UserId string `json:"userId"`
}

type UpdatePasswordRequest struct {
	// confirm new password
	ConfirmPassword string `json:"confirmPassword"`
	// the new password
	NewPassword string `json:"newPassword"`
	// the old password
	OldPassword string `json:"oldPassword"`
	// the account id
	UserId string `json:"userId"`
}

type UpdatePasswordResponse struct {
}

type UpdateRequest struct {
	// the new email address
	Email string `json:"email"`
	// the account id
	Id string `json:"id"`
	// the user profile as map<string,string>
	Profile map[string]string `json:"profile"`
	// the new username
	Username string `json:"username"`
}

type UpdateResponse struct {
}

type VerifyEmailRequest struct {
	// The token from the verification email
	Token string `json:"token"`
}

type VerifyEmailResponse struct {
}
