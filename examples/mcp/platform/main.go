// Platform example: AI agents interacting with a real microservices platform.
//
// This example mirrors the micro/blog platform (https://github.com/micro/blog)
// — a microblogging platform built on Go Micro with Users, Posts, Comments,
// and Mail services. It demonstrates how existing microservices become
// AI-accessible through MCP with zero changes to business logic.
//
// The services run as a single binary for convenience. In production,
// each would be a separate process discovered via the registry.
//
// Run:
//
//	go run .
//
// MCP tools: http://localhost:3001/mcp/tools
//
// Agent scenarios:
//
//	"Sign me up as alice with password secret123"
//	"Log in as alice and write a blog post about Go concurrency"
//	"List all posts and comment on the first one"
//	"Send a welcome email to alice"
//	"Tag the Go concurrency post with 'golang' and 'tutorial'"
//	"Show me alice's profile and all her posts"
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"go-micro.dev/v5"
	"go-micro.dev/v5/gateway/mcp"
	"go-micro.dev/v5/server"
)

// ---------------------------------------------------------------------------
// Users service — account registration, login, profiles
// ---------------------------------------------------------------------------

type User struct {
	ID        string `json:"id" description:"Unique user identifier"`
	Name      string `json:"name" description:"Display name"`
	Status    string `json:"status" description:"Bio or status message"`
	CreatedAt int64  `json:"created_at" description:"Unix timestamp of account creation"`
}

type SignupRequest struct {
	Name     string `json:"name" description:"Username (required, 3-20 characters)"`
	Password string `json:"password" description:"Password (required, minimum 6 characters)"`
}
type SignupResponse struct {
	User  *User  `json:"user" description:"The newly created user account"`
	Token string `json:"token" description:"Session token for authenticated requests"`
}

type LoginRequest struct {
	Name     string `json:"name" description:"Username"`
	Password string `json:"password" description:"Password"`
}
type LoginResponse struct {
	User  *User  `json:"user" description:"The authenticated user"`
	Token string `json:"token" description:"Session token for authenticated requests"`
}

type GetProfileRequest struct {
	ID string `json:"id" description:"User ID to look up"`
}
type GetProfileResponse struct {
	User *User `json:"user" description:"The user profile"`
}

type UpdateStatusRequest struct {
	ID     string `json:"id" description:"User ID"`
	Status string `json:"status" description:"New bio or status message"`
}
type UpdateStatusResponse struct {
	User *User `json:"user" description:"Updated user profile"`
}

type ListUsersRequest struct{}
type ListUsersResponse struct {
	Users []*User `json:"users" description:"All registered users"`
}

type Users struct {
	mu        sync.RWMutex
	users     map[string]*User
	passwords map[string]string // name -> password (plaintext for demo only)
	tokens    map[string]string // token -> user ID
	nextID    int
}

func NewUsers() *Users {
	return &Users{
		users:     make(map[string]*User),
		passwords: make(map[string]string),
		tokens:    make(map[string]string),
	}
}

// Signup creates a new user account and returns a session token.
// The username must be unique. Use the returned token for authenticated operations.
//
// @example {"name": "alice", "password": "secret123"}
func (s *Users) Signup(ctx context.Context, req *SignupRequest, rsp *SignupResponse) error {
	if req.Name == "" || len(req.Name) < 3 {
		return fmt.Errorf("name must be at least 3 characters")
	}
	if req.Password == "" || len(req.Password) < 6 {
		return fmt.Errorf("password must be at least 6 characters")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check uniqueness
	for _, u := range s.users {
		if strings.EqualFold(u.Name, req.Name) {
			return fmt.Errorf("username %q is already taken", req.Name)
		}
	}

	s.nextID++
	user := &User{
		ID:        fmt.Sprintf("user-%d", s.nextID),
		Name:      req.Name,
		CreatedAt: time.Now().Unix(),
	}
	s.users[user.ID] = user
	s.passwords[req.Name] = req.Password

	token := generateToken()
	s.tokens[token] = user.ID

	rsp.User = user
	rsp.Token = token
	return nil
}

// Login authenticates a user and returns a session token.
// Returns an error if the credentials are invalid.
//
// @example {"name": "alice", "password": "secret123"}
func (s *Users) Login(ctx context.Context, req *LoginRequest, rsp *LoginResponse) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pass, ok := s.passwords[req.Name]
	if !ok || pass != req.Password {
		return fmt.Errorf("invalid username or password")
	}

	// Find user by name
	for _, u := range s.users {
		if u.Name == req.Name {
			token := generateToken()
			s.tokens[token] = u.ID
			rsp.User = u
			rsp.Token = token
			return nil
		}
	}
	return fmt.Errorf("user not found")
}

// GetProfile retrieves a user's public profile by ID.
//
// @example {"id": "user-1"}
func (s *Users) GetProfile(ctx context.Context, req *GetProfileRequest, rsp *GetProfileResponse) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, ok := s.users[req.ID]
	if !ok {
		return fmt.Errorf("user %s not found", req.ID)
	}
	rsp.User = u
	return nil
}

// UpdateStatus sets a user's bio or status message.
//
// @example {"id": "user-1", "status": "Writing about Go and microservices"}
func (s *Users) UpdateStatus(ctx context.Context, req *UpdateStatusRequest, rsp *UpdateStatusResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	u, ok := s.users[req.ID]
	if !ok {
		return fmt.Errorf("user %s not found", req.ID)
	}
	u.Status = req.Status
	rsp.User = u
	return nil
}

// List returns all registered users on the platform.
//
// @example {}
func (s *Users) List(ctx context.Context, req *ListUsersRequest, rsp *ListUsersResponse) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, u := range s.users {
		rsp.Users = append(rsp.Users, u)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Posts service — blog posts with markdown and tags
// ---------------------------------------------------------------------------

type Post struct {
	ID         string   `json:"id" description:"Unique post identifier"`
	Title      string   `json:"title" description:"Post title"`
	Content    string   `json:"content" description:"Post body in markdown"`
	AuthorID   string   `json:"author_id" description:"ID of the post author"`
	AuthorName string   `json:"author_name" description:"Display name of the author"`
	Tags       []string `json:"tags,omitempty" description:"Post tags for categorization"`
	CreatedAt  int64    `json:"created_at" description:"Unix timestamp of creation"`
	UpdatedAt  int64    `json:"updated_at" description:"Unix timestamp of last update"`
}

type CreatePostRequest struct {
	Title      string `json:"title" description:"Post title (required)"`
	Content    string `json:"content" description:"Post body in markdown (required)"`
	AuthorID   string `json:"author_id" description:"Author's user ID (required)"`
	AuthorName string `json:"author_name" description:"Author's display name (required)"`
}
type CreatePostResponse struct {
	Post *Post `json:"post" description:"The newly created post"`
}

type ReadPostRequest struct {
	ID string `json:"id" description:"Post ID to retrieve"`
}
type ReadPostResponse struct {
	Post *Post `json:"post" description:"The requested post"`
}

type UpdatePostRequest struct {
	ID      string `json:"id" description:"Post ID to update (required)"`
	Title   string `json:"title" description:"New title"`
	Content string `json:"content" description:"New content in markdown"`
}
type UpdatePostResponse struct {
	Post *Post `json:"post" description:"The updated post"`
}

type DeletePostRequest struct {
	ID string `json:"id" description:"Post ID to delete"`
}
type DeletePostResponse struct {
	Message string `json:"message" description:"Confirmation message"`
}

type ListPostsRequest struct {
	AuthorID string `json:"author_id,omitempty" description:"Filter by author ID (optional)"`
}
type ListPostsResponse struct {
	Posts []*Post `json:"posts" description:"Posts in reverse chronological order"`
	Total int     `json:"total" description:"Total number of matching posts"`
}

type TagPostRequest struct {
	PostID string `json:"post_id" description:"Post to tag"`
	Tag    string `json:"tag" description:"Tag to add (lowercase, no spaces)"`
}
type TagPostResponse struct {
	Post *Post `json:"post" description:"Post with updated tags"`
}

type UntagPostRequest struct {
	PostID string `json:"post_id" description:"Post to untag"`
	Tag    string `json:"tag" description:"Tag to remove"`
}
type UntagPostResponse struct {
	Post *Post `json:"post" description:"Post with updated tags"`
}

type ListTagsRequest struct{}
type ListTagsResponse struct {
	Tags []string `json:"tags" description:"All tags in use, sorted alphabetically"`
}

type Posts struct {
	mu     sync.RWMutex
	posts  map[string]*Post
	nextID int
}

func NewPosts() *Posts {
	return &Posts{posts: make(map[string]*Post)}
}

// Create publishes a new blog post. Title, content, author_id, and author_name
// are required. Content supports markdown formatting.
//
// @example {"title": "Getting Started with Go Micro", "content": "Go Micro makes it easy to build microservices...", "author_id": "user-1", "author_name": "alice"}
func (s *Posts) Create(ctx context.Context, req *CreatePostRequest, rsp *CreatePostResponse) error {
	if req.Title == "" {
		return fmt.Errorf("title is required")
	}
	if req.Content == "" {
		return fmt.Errorf("content is required")
	}
	if req.AuthorID == "" {
		return fmt.Errorf("author_id is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	now := time.Now().Unix()
	post := &Post{
		ID:         fmt.Sprintf("post-%d", s.nextID),
		Title:      req.Title,
		Content:    req.Content,
		AuthorID:   req.AuthorID,
		AuthorName: req.AuthorName,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	s.posts[post.ID] = post
	rsp.Post = post
	return nil
}

// Read retrieves a single blog post by ID.
//
// @example {"id": "post-1"}
func (s *Posts) Read(ctx context.Context, req *ReadPostRequest, rsp *ReadPostResponse) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.posts[req.ID]
	if !ok {
		return fmt.Errorf("post %s not found", req.ID)
	}
	rsp.Post = p
	return nil
}

// Update modifies a blog post's title and/or content.
// Only non-empty fields are updated.
//
// @example {"id": "post-1", "title": "Updated Title", "content": "New content here..."}
func (s *Posts) Update(ctx context.Context, req *UpdatePostRequest, rsp *UpdatePostResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.posts[req.ID]
	if !ok {
		return fmt.Errorf("post %s not found", req.ID)
	}
	if req.Title != "" {
		p.Title = req.Title
	}
	if req.Content != "" {
		p.Content = req.Content
	}
	p.UpdatedAt = time.Now().Unix()
	rsp.Post = p
	return nil
}

// Delete removes a blog post permanently.
//
// @example {"id": "post-1"}
func (s *Posts) Delete(ctx context.Context, req *DeletePostRequest, rsp *DeletePostResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.posts[req.ID]; !ok {
		return fmt.Errorf("post %s not found", req.ID)
	}
	delete(s.posts, req.ID)
	rsp.Message = fmt.Sprintf("post %s deleted", req.ID)
	return nil
}

// List returns blog posts in reverse chronological order.
// Optionally filter by author_id to see a specific user's posts.
//
// @example {"author_id": "user-1"}
func (s *Posts) List(ctx context.Context, req *ListPostsRequest, rsp *ListPostsResponse) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, p := range s.posts {
		if req.AuthorID != "" && p.AuthorID != req.AuthorID {
			continue
		}
		rsp.Posts = append(rsp.Posts, p)
	}
	sort.Slice(rsp.Posts, func(i, j int) bool {
		return rsp.Posts[i].CreatedAt > rsp.Posts[j].CreatedAt
	})
	rsp.Total = len(rsp.Posts)
	return nil
}

// TagPost adds a tag to a post. Tags are useful for categorization
// and discovery. Duplicate tags are ignored.
//
// @example {"post_id": "post-1", "tag": "golang"}
func (s *Posts) TagPost(ctx context.Context, req *TagPostRequest, rsp *TagPostResponse) error {
	if req.PostID == "" || req.Tag == "" {
		return fmt.Errorf("post_id and tag are required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.posts[req.PostID]
	if !ok {
		return fmt.Errorf("post %s not found", req.PostID)
	}

	tag := strings.ToLower(strings.TrimSpace(req.Tag))
	for _, t := range p.Tags {
		if t == tag {
			rsp.Post = p
			return nil
		}
	}
	p.Tags = append(p.Tags, tag)
	p.UpdatedAt = time.Now().Unix()
	rsp.Post = p
	return nil
}

// UntagPost removes a tag from a post.
//
// @example {"post_id": "post-1", "tag": "golang"}
func (s *Posts) UntagPost(ctx context.Context, req *UntagPostRequest, rsp *UntagPostResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.posts[req.PostID]
	if !ok {
		return fmt.Errorf("post %s not found", req.PostID)
	}

	filtered := make([]string, 0, len(p.Tags))
	for _, t := range p.Tags {
		if t != req.Tag {
			filtered = append(filtered, t)
		}
	}
	p.Tags = filtered
	p.UpdatedAt = time.Now().Unix()
	rsp.Post = p
	return nil
}

// ListTags returns all tags currently in use across all posts.
//
// @example {}
func (s *Posts) ListTags(ctx context.Context, req *ListTagsRequest, rsp *ListTagsResponse) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	seen := make(map[string]bool)
	for _, p := range s.posts {
		for _, t := range p.Tags {
			seen[t] = true
		}
	}
	for t := range seen {
		rsp.Tags = append(rsp.Tags, t)
	}
	sort.Strings(rsp.Tags)
	return nil
}

// ---------------------------------------------------------------------------
// Comments service — threaded comments on posts
// ---------------------------------------------------------------------------

type Comment struct {
	ID         string `json:"id" description:"Unique comment identifier"`
	PostID     string `json:"post_id" description:"ID of the post this comment belongs to"`
	Content    string `json:"content" description:"Comment text"`
	AuthorID   string `json:"author_id" description:"ID of the comment author"`
	AuthorName string `json:"author_name" description:"Display name of the author"`
	CreatedAt  int64  `json:"created_at" description:"Unix timestamp of creation"`
}

type CreateCommentRequest struct {
	PostID     string `json:"post_id" description:"Post to comment on (required)"`
	Content    string `json:"content" description:"Comment text (required)"`
	AuthorID   string `json:"author_id" description:"Author's user ID (required)"`
	AuthorName string `json:"author_name" description:"Author's display name (required)"`
}
type CreateCommentResponse struct {
	Comment *Comment `json:"comment" description:"The newly created comment"`
}

type ListCommentsRequest struct {
	PostID   string `json:"post_id,omitempty" description:"Filter by post ID (optional)"`
	AuthorID string `json:"author_id,omitempty" description:"Filter by author ID (optional)"`
}
type ListCommentsResponse struct {
	Comments []*Comment `json:"comments" description:"Matching comments"`
}

type DeleteCommentRequest struct {
	ID string `json:"id" description:"Comment ID to delete"`
}
type DeleteCommentResponse struct {
	Message string `json:"message" description:"Confirmation message"`
}

type Comments struct {
	mu       sync.RWMutex
	comments []*Comment
	nextID   int
}

// Create adds a comment to a blog post. Post ID, content, author_id,
// and author_name are all required.
//
// @example {"post_id": "post-1", "content": "Great article! Very helpful.", "author_id": "user-2", "author_name": "bob"}
func (s *Comments) Create(ctx context.Context, req *CreateCommentRequest, rsp *CreateCommentResponse) error {
	if req.PostID == "" {
		return fmt.Errorf("post_id is required")
	}
	if req.Content == "" {
		return fmt.Errorf("content is required")
	}
	if req.AuthorID == "" {
		return fmt.Errorf("author_id is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	comment := &Comment{
		ID:         fmt.Sprintf("comment-%d", s.nextID),
		PostID:     req.PostID,
		Content:    req.Content,
		AuthorID:   req.AuthorID,
		AuthorName: req.AuthorName,
		CreatedAt:  time.Now().Unix(),
	}
	s.comments = append(s.comments, comment)
	rsp.Comment = comment
	return nil
}

// List returns comments, optionally filtered by post or author.
// Use post_id to get all comments on a specific post.
//
// @example {"post_id": "post-1"}
func (s *Comments) List(ctx context.Context, req *ListCommentsRequest, rsp *ListCommentsResponse) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, c := range s.comments {
		if req.PostID != "" && c.PostID != req.PostID {
			continue
		}
		if req.AuthorID != "" && c.AuthorID != req.AuthorID {
			continue
		}
		rsp.Comments = append(rsp.Comments, c)
	}
	return nil
}

// Delete removes a comment by ID.
//
// @example {"id": "comment-1"}
func (s *Comments) Delete(ctx context.Context, req *DeleteCommentRequest, rsp *DeleteCommentResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, c := range s.comments {
		if c.ID == req.ID {
			s.comments = append(s.comments[:i], s.comments[i+1:]...)
			rsp.Message = fmt.Sprintf("comment %s deleted", req.ID)
			return nil
		}
	}
	return fmt.Errorf("comment %s not found", req.ID)
}

// ---------------------------------------------------------------------------
// Mail service — internal messaging between users
// ---------------------------------------------------------------------------

type MailMessage struct {
	ID        string `json:"id" description:"Unique message identifier"`
	From      string `json:"from" description:"Sender username"`
	To        string `json:"to" description:"Recipient username"`
	Subject   string `json:"subject" description:"Message subject line"`
	Body      string `json:"body" description:"Message body text"`
	Read      bool   `json:"read" description:"Whether the message has been read"`
	CreatedAt int64  `json:"created_at" description:"Unix timestamp of when the message was sent"`
}

type SendMailRequest struct {
	From    string `json:"from" description:"Sender username (required)"`
	To      string `json:"to" description:"Recipient username (required)"`
	Subject string `json:"subject" description:"Message subject (required)"`
	Body    string `json:"body" description:"Message body (required)"`
}
type SendMailResponse struct {
	Message *MailMessage `json:"message" description:"The sent message"`
}

type ReadMailRequest struct {
	User string `json:"user" description:"Username to read inbox for"`
}
type ReadMailResponse struct {
	Messages []*MailMessage `json:"messages" description:"Inbox messages, newest first"`
}

type Mail struct {
	mu       sync.RWMutex
	messages []*MailMessage
	nextID   int
}

// Send delivers a message to another user on the platform.
// Both sender and recipient are identified by username.
//
// @example {"from": "alice", "to": "bob", "subject": "Welcome!", "body": "Hey Bob, welcome to the platform!"}
func (s *Mail) Send(ctx context.Context, req *SendMailRequest, rsp *SendMailResponse) error {
	if req.From == "" || req.To == "" {
		return fmt.Errorf("from and to are required")
	}
	if req.Subject == "" {
		return fmt.Errorf("subject is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	msg := &MailMessage{
		ID:        fmt.Sprintf("mail-%d", s.nextID),
		From:      req.From,
		To:        req.To,
		Subject:   req.Subject,
		Body:      req.Body,
		CreatedAt: time.Now().Unix(),
	}
	s.messages = append(s.messages, msg)
	rsp.Message = msg
	return nil
}

// Read returns all messages in a user's inbox, newest first.
//
// @example {"user": "alice"}
func (s *Mail) Read(ctx context.Context, req *ReadMailRequest, rsp *ReadMailResponse) error {
	if req.User == "" {
		return fmt.Errorf("user is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for i := len(s.messages) - 1; i >= 0; i-- {
		if s.messages[i].To == req.User {
			s.messages[i].Read = true
			rsp.Messages = append(rsp.Messages, s.messages[i])
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Main — wire up all services with MCP gateway
// ---------------------------------------------------------------------------

func main() {
	service := micro.New("platform",
		micro.Address(":9090"),
		mcp.WithMCP(":3001"),
	)
	service.Init()

	users := NewUsers()
	posts := NewPosts()

	// Seed some demo data so agents have something to work with
	seedData(users, posts)

	service.Handle(users)
	service.Handle(posts)
	service.Handle(&Comments{})
	service.Handle(&Mail{},
		server.WithEndpointScopes("Mail.Send", "mail:write"),
		server.WithEndpointScopes("Mail.Read", "mail:read"),
	)

	printBanner()

	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}

func seedData(users *Users, posts *Posts) {
	// Create demo users
	var aliceRsp SignupResponse
	users.Signup(context.Background(), &SignupRequest{
		Name: "alice", Password: "secret123",
	}, &aliceRsp)

	var bobRsp SignupResponse
	users.Signup(context.Background(), &SignupRequest{
		Name: "bob", Password: "secret123",
	}, &bobRsp)

	// Alice writes a welcome post
	var postRsp CreatePostResponse
	posts.Create(context.Background(), &CreatePostRequest{
		Title:      "Welcome to the Platform",
		Content:    "This is the first post on our new blogging platform. Built with Go Micro, every service is automatically accessible to AI agents through MCP.",
		AuthorID:   aliceRsp.User.ID,
		AuthorName: "alice",
	}, &postRsp)

	// Tag it
	posts.TagPost(context.Background(), &TagPostRequest{
		PostID: postRsp.Post.ID, Tag: "welcome",
	}, &TagPostResponse{})
	posts.TagPost(context.Background(), &TagPostRequest{
		PostID: postRsp.Post.ID, Tag: "go-micro",
	}, &TagPostResponse{})
}

func printBanner() {
	fmt.Println()
	fmt.Println("  Platform Demo — AI-Native Microservices")
	fmt.Println()
	fmt.Println("  Services:   Users, Posts, Comments, Mail")
	fmt.Println("  MCP Tools:  http://localhost:3001/mcp/tools")
	fmt.Println("  RPC:        localhost:9090")
	fmt.Println()
	fmt.Println("  Seeded:     alice (user-1), bob (user-2)")
	fmt.Println("              1 post with tags [welcome, go-micro]")
	fmt.Println()
	fmt.Println("  Try asking an agent:")
	fmt.Println()
	fmt.Println(`    "Sign up a new user called carol"`)
	fmt.Println(`    "Log in as alice and write a post about Go concurrency patterns"`)
	fmt.Println(`    "List all posts and comment on the welcome post as bob"`)
	fmt.Println(`    "Tag alice's post with 'tutorial' and 'golang'"`)
	fmt.Println(`    "Send a mail from alice to bob welcoming him to the platform"`)
	fmt.Println(`    "Show me bob's inbox"`)
	fmt.Println(`    "List all users and show me all tags in use"`)
	fmt.Println()
}

func generateToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
