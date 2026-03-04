// Agent Demo — A multi-service project management app
//
// This example shows three Go Micro services (projects, tasks, team)
// working together through the MCP gateway, letting an AI agent
// manage projects using natural language.
//
// Run:
//
//	go run main.go
//
// Then open the agent playground at http://localhost:8080/agent
// or connect Claude Code via: micro mcp serve
package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"go-micro.dev/v5"
	"go-micro.dev/v5/gateway/mcp"
	"go-micro.dev/v5/server"
)

// ---------------------------------------------------------------------------
// Projects service
// ---------------------------------------------------------------------------

type Project struct {
	ID          string    `json:"id" description:"Unique project identifier"`
	Name        string    `json:"name" description:"Project name"`
	Description string    `json:"description" description:"What the project is about"`
	Status      string    `json:"status" description:"Project status: planning, active, or completed"`
	CreatedAt   time.Time `json:"created_at" description:"When the project was created"`
}

type CreateProjectRequest struct {
	Name        string `json:"name" description:"Project name (required)"`
	Description string `json:"description" description:"Short description of the project"`
}

type CreateProjectResponse struct {
	Project *Project `json:"project" description:"The newly created project"`
}

type GetProjectRequest struct {
	ID string `json:"id" description:"Project ID to retrieve"`
}

type GetProjectResponse struct {
	Project *Project `json:"project" description:"The requested project"`
}

type ListProjectsRequest struct {
	Status string `json:"status,omitempty" description:"Filter by status: planning, active, completed (optional)"`
}

type ListProjectsResponse struct {
	Projects []*Project `json:"projects" description:"List of matching projects"`
}

type ProjectService struct {
	mu       sync.RWMutex
	projects map[string]*Project
	nextID   int
}

// Create creates a new project with the given name and description.
// Returns the project with a generated ID and initial status of "planning".
//
// @example {"name": "Website Redesign", "description": "Redesign the company website with new branding"}
func (s *ProjectService) Create(ctx context.Context, req *CreateProjectRequest, rsp *CreateProjectResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	p := &Project{
		ID:          fmt.Sprintf("proj-%d", s.nextID),
		Name:        req.Name,
		Description: req.Description,
		Status:      "planning",
		CreatedAt:   time.Now(),
	}
	s.projects[p.ID] = p
	rsp.Project = p
	return nil
}

// Get retrieves a project by ID.
// Returns an error if the project does not exist.
//
// @example {"id": "proj-1"}
func (s *ProjectService) Get(ctx context.Context, req *GetProjectRequest, rsp *GetProjectResponse) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.projects[req.ID]
	if !ok {
		return fmt.Errorf("project %s not found", req.ID)
	}
	rsp.Project = p
	return nil
}

// List returns all projects, optionally filtered by status.
// Valid status values: planning, active, completed.
//
// @example {"status": "active"}
func (s *ProjectService) List(ctx context.Context, req *ListProjectsRequest, rsp *ListProjectsResponse) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.projects {
		if req.Status == "" || p.Status == req.Status {
			rsp.Projects = append(rsp.Projects, p)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Tasks service
// ---------------------------------------------------------------------------

type Task struct {
	ID        string `json:"id" description:"Unique task identifier"`
	ProjectID string `json:"project_id" description:"ID of the project this task belongs to"`
	Title     string `json:"title" description:"Short task title"`
	Status    string `json:"status" description:"Task status: todo, in_progress, or done"`
	Assignee  string `json:"assignee,omitempty" description:"Username of the person assigned"`
	Priority  string `json:"priority" description:"Priority: low, medium, or high"`
}

type CreateTaskRequest struct {
	ProjectID string `json:"project_id" description:"Project ID to add the task to (required)"`
	Title     string `json:"title" description:"Task title (required)"`
	Assignee  string `json:"assignee,omitempty" description:"Username to assign (optional)"`
	Priority  string `json:"priority,omitempty" description:"Priority: low, medium, or high (default: medium)"`
}

type CreateTaskResponse struct {
	Task *Task `json:"task" description:"The newly created task"`
}

type ListTasksRequest struct {
	ProjectID string `json:"project_id,omitempty" description:"Filter by project ID (optional)"`
	Assignee  string `json:"assignee,omitempty" description:"Filter by assignee username (optional)"`
	Status    string `json:"status,omitempty" description:"Filter by status: todo, in_progress, done (optional)"`
}

type ListTasksResponse struct {
	Tasks []*Task `json:"tasks" description:"List of matching tasks"`
}

type UpdateTaskRequest struct {
	ID       string `json:"id" description:"Task ID to update"`
	Status   string `json:"status,omitempty" description:"New status: todo, in_progress, or done"`
	Assignee string `json:"assignee,omitempty" description:"New assignee username"`
}

type UpdateTaskResponse struct {
	Task *Task `json:"task" description:"The updated task"`
}

type TaskService struct {
	mu     sync.RWMutex
	tasks  map[string]*Task
	nextID int
}

// Create creates a new task in a project.
// Returns the task with a generated ID, initial status of "todo", and default priority of "medium".
//
// @example {"project_id": "proj-1", "title": "Design homepage mockup", "assignee": "alice", "priority": "high"}
func (s *TaskService) Create(ctx context.Context, req *CreateTaskRequest, rsp *CreateTaskResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	priority := req.Priority
	if priority == "" {
		priority = "medium"
	}
	t := &Task{
		ID:        fmt.Sprintf("task-%d", s.nextID),
		ProjectID: req.ProjectID,
		Title:     req.Title,
		Status:    "todo",
		Assignee:  req.Assignee,
		Priority:  priority,
	}
	s.tasks[t.ID] = t
	rsp.Task = t
	return nil
}

// List returns tasks filtered by project, assignee, or status.
// All filters are optional; omit all to list every task.
//
// @example {"project_id": "proj-1", "status": "todo"}
func (s *TaskService) List(ctx context.Context, req *ListTasksRequest, rsp *ListTasksResponse) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, t := range s.tasks {
		if req.ProjectID != "" && t.ProjectID != req.ProjectID {
			continue
		}
		if req.Assignee != "" && t.Assignee != req.Assignee {
			continue
		}
		if req.Status != "" && t.Status != req.Status {
			continue
		}
		rsp.Tasks = append(rsp.Tasks, t)
	}
	return nil
}

// Update modifies a task's status or assignee.
// Only provided fields are changed; omitted fields stay the same.
// Returns an error if the task does not exist.
//
// @example {"id": "task-1", "status": "in_progress"}
func (s *TaskService) Update(ctx context.Context, req *UpdateTaskRequest, rsp *UpdateTaskResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[req.ID]
	if !ok {
		return fmt.Errorf("task %s not found", req.ID)
	}
	if req.Status != "" {
		t.Status = req.Status
	}
	if req.Assignee != "" {
		t.Assignee = req.Assignee
	}
	rsp.Task = t
	return nil
}

// ---------------------------------------------------------------------------
// Team service
// ---------------------------------------------------------------------------

type Member struct {
	Username string   `json:"username" description:"Unique username"`
	Name     string   `json:"name" description:"Display name"`
	Role     string   `json:"role" description:"Role: engineer, designer, or manager"`
	Skills   []string `json:"skills" description:"List of skills (e.g. go, react, figma)"`
}

type AddMemberRequest struct {
	Username string   `json:"username" description:"Unique username (required)"`
	Name     string   `json:"name" description:"Display name (required)"`
	Role     string   `json:"role" description:"Role: engineer, designer, or manager"`
	Skills   []string `json:"skills,omitempty" description:"List of skills"`
}

type AddMemberResponse struct {
	Member *Member `json:"member" description:"The added team member"`
}

type ListMembersRequest struct {
	Role  string `json:"role,omitempty" description:"Filter by role: engineer, designer, manager (optional)"`
	Skill string `json:"skill,omitempty" description:"Filter by skill (optional, e.g. 'go' or 'react')"`
}

type ListMembersResponse struct {
	Members []*Member `json:"members" description:"List of matching team members"`
}

type GetMemberRequest struct {
	Username string `json:"username" description:"Username to look up"`
}

type GetMemberResponse struct {
	Member *Member `json:"member" description:"The team member"`
}

type TeamService struct {
	mu      sync.RWMutex
	members map[string]*Member
}

// Add adds a new team member.
// Returns the member with their assigned role and skills.
//
// @example {"username": "alice", "name": "Alice Chen", "role": "engineer", "skills": ["go", "react"]}
func (s *TeamService) Add(ctx context.Context, req *AddMemberRequest, rsp *AddMemberResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	m := &Member{
		Username: req.Username,
		Name:     req.Name,
		Role:     req.Role,
		Skills:   req.Skills,
	}
	s.members[m.Username] = m
	rsp.Member = m
	return nil
}

// List returns team members, optionally filtered by role or skill.
//
// @example {"role": "engineer"}
func (s *TeamService) List(ctx context.Context, req *ListMembersRequest, rsp *ListMembersResponse) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, m := range s.members {
		if req.Role != "" && m.Role != req.Role {
			continue
		}
		if req.Skill != "" && !hasSkill(m.Skills, req.Skill) {
			continue
		}
		rsp.Members = append(rsp.Members, m)
	}
	return nil
}

// Get retrieves a team member by username.
// Returns an error if the member does not exist.
//
// @example {"username": "alice"}
func (s *TeamService) Get(ctx context.Context, req *GetMemberRequest, rsp *GetMemberResponse) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.members[req.Username]
	if !ok {
		return fmt.Errorf("member %s not found", req.Username)
	}
	rsp.Member = m
	return nil
}

func hasSkill(skills []string, target string) bool {
	for _, s := range skills {
		if strings.EqualFold(s, target) {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Main — wire everything together
// ---------------------------------------------------------------------------

func main() {
	// Create the service
	service := micro.New("demo",
		micro.Address(":9090"),
		// Start MCP gateway alongside the service
		mcp.WithMCP(":3000"),
	)
	service.Init()

	// Register all three handlers with scopes
	service.Handle(
		&ProjectService{projects: make(map[string]*Project)},
		server.WithEndpointScopes("ProjectService.Create", "projects:write"),
		server.WithEndpointScopes("ProjectService.Get", "projects:read"),
		server.WithEndpointScopes("ProjectService.List", "projects:read"),
	)

	service.Handle(
		&TaskService{tasks: make(map[string]*Task)},
		server.WithEndpointScopes("TaskService.Create", "tasks:write"),
		server.WithEndpointScopes("TaskService.List", "tasks:read"),
		server.WithEndpointScopes("TaskService.Update", "tasks:write"),
	)

	service.Handle(
		&TeamService{members: make(map[string]*Member)},
		server.WithEndpointScopes("TeamService.Add", "team:write"),
		server.WithEndpointScopes("TeamService.List", "team:read"),
		server.WithEndpointScopes("TeamService.Get", "team:read"),
	)

	// Seed some demo data
	seedData(service.Server())

	fmt.Println()
	fmt.Println("  Agent Demo")
	fmt.Println()
	fmt.Println("  MCP Gateway   http://localhost:3000")
	fmt.Println("  MCP Tools     http://localhost:3000/mcp/tools")
	fmt.Println("  WebSocket     ws://localhost:3000/mcp/ws")
	fmt.Println()
	fmt.Println("  Try these prompts with Claude Code or the agent playground:")
	fmt.Println()
	fmt.Println("    \"What projects do we have?\"")
	fmt.Println("    \"Create a task for alice to design the new landing page\"")
	fmt.Println("    \"Show me all high-priority tasks that are still todo\"")
	fmt.Println("    \"Who on the team knows React?\"")
	fmt.Println("    \"Give me a status update on the Website Redesign project\"")
	fmt.Println()

	service.Run()
}

// seedData pre-populates the services with realistic demo data.
func seedData(srv server.Server) {
	ctx := context.Background()

	// Seed team members
	team := &TeamService{members: make(map[string]*Member)}
	for _, m := range []AddMemberRequest{
		{Username: "alice", Name: "Alice Chen", Role: "engineer", Skills: []string{"go", "grpc", "kubernetes"}},
		{Username: "bob", Name: "Bob Park", Role: "designer", Skills: []string{"figma", "css", "react"}},
		{Username: "charlie", Name: "Charlie Kim", Role: "engineer", Skills: []string{"go", "react", "postgres"}},
		{Username: "diana", Name: "Diana Flores", Role: "manager", Skills: []string{"project-management", "scrum"}},
	} {
		req := m
		team.Add(ctx, &req, &AddMemberResponse{})
	}

	// Seed projects
	projects := &ProjectService{projects: make(map[string]*Project)}
	projects.Create(ctx, &CreateProjectRequest{
		Name:        "Website Redesign",
		Description: "Redesign the company website with new branding and improved UX",
	}, &CreateProjectResponse{})
	projects.projects["proj-1"].Status = "active"

	projects.Create(ctx, &CreateProjectRequest{
		Name:        "API v2 Migration",
		Description: "Migrate all services from REST to gRPC with backward compatibility",
	}, &CreateProjectResponse{})
	projects.projects["proj-2"].Status = "planning"

	// Seed tasks
	tasks := &TaskService{tasks: make(map[string]*Task)}
	for _, t := range []CreateTaskRequest{
		{ProjectID: "proj-1", Title: "Design new homepage layout", Assignee: "bob", Priority: "high"},
		{ProjectID: "proj-1", Title: "Implement responsive nav component", Assignee: "charlie", Priority: "high"},
		{ProjectID: "proj-1", Title: "Write copy for about page", Priority: "medium"},
		{ProjectID: "proj-1", Title: "Set up CI/CD for new site", Assignee: "alice", Priority: "medium"},
		{ProjectID: "proj-2", Title: "Audit existing REST endpoints", Assignee: "alice", Priority: "high"},
		{ProjectID: "proj-2", Title: "Design gRPC proto files", Priority: "medium"},
		{ProjectID: "proj-2", Title: "Write migration guide", Assignee: "diana", Priority: "low"},
	} {
		req := t
		tasks.Create(ctx, &req, &CreateTaskResponse{})
	}
	// Mark a couple tasks as in_progress
	tasks.tasks["task-1"].Status = "in_progress"
	tasks.tasks["task-5"].Status = "in_progress"

	// Register the seeded handlers (replace the empty ones registered above)
	// Note: in a real app these would be separate services. Here we register
	// pre-seeded instances so the demo starts with data.
	srv.Handle(srv.NewHandler(projects))
	srv.Handle(srv.NewHandler(tasks))
	srv.Handle(srv.NewHandler(team))
}
