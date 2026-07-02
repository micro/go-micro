// Agent Ollama — a self-contained agent powered by Ollama Cloud.
//
// This example demonstrates the full harness loop — service tools, custom
// tools, agent memory, guardrails, and streaming — using the Ollama
// provider with gemma4:31b-cloud on Ollama Cloud.
//
// It creates a "knowledge" service with two endpoints (Add, Search) that
// the agent discovers as tools, plus a custom "current_time" tool. The
// agent answers natural-language questions by calling those tools.
//
// Run (Ollama Cloud — default):
//
//	OLLAMA_API_KEY=your-key go run main.go
//
// Run (local Ollama):
//
//	OLLAMA_BASE_URL=http://localhost:11434 \
//	OLLAMA_MODEL=llama3.2 \
//	go run main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"go-micro.dev/v6"
	"go-micro.dev/v6/agent"
)

// ---------------------------------------------------------------------------
// knowledge service — a tiny in-memory knowledge base
// ---------------------------------------------------------------------------

type KnowledgeEntry struct {
	ID      string `json:"id" description:"Unique entry identifier"`
	Topic   string `json:"topic" description:"Topic or category"`
	Content string `json:"content" description:"The knowledge content"`
}

type AddKnowledgeRequest struct {
	Topic   string `json:"topic" description:"Topic or category (required)"`
	Content string `json:"content" description:"The knowledge content (required)"`
}

type AddKnowledgeResponse struct {
	Entry *KnowledgeEntry `json:"entry" description:"The added entry"`
}

type SearchKnowledgeRequest struct {
	Topic   string `json:"topic,omitempty" description:"Filter by topic (optional)"`
	Keyword string `json:"keyword,omitempty" description:"Search keyword in content (optional)"`
}

type SearchKnowledgeResponse struct {
	Entries []*KnowledgeEntry `json:"entries" description:"Matching entries"`
}

type KnowledgeService struct {
	mu      sync.RWMutex
	entries []*KnowledgeEntry
	nextID  int
}

// Add stores a new knowledge entry.
//
// @example {"topic": "go", "content": "Go interfaces are implicit."}
func (s *KnowledgeService) Add(ctx context.Context, req *AddKnowledgeRequest, rsp *AddKnowledgeResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	e := &KnowledgeEntry{
		ID:      fmt.Sprintf("kb-%d", s.nextID),
		Topic:   req.Topic,
		Content: req.Content,
	}
	s.entries = append(s.entries, e)
	rsp.Entry = e
	return nil
}

// Search finds knowledge entries by topic or keyword.
//
// @example {"topic": "go"}
// @example {"keyword": "interface"}
func (s *KnowledgeService) Search(ctx context.Context, req *SearchKnowledgeRequest, rsp *SearchKnowledgeResponse) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, e := range s.entries {
		if req.Topic != "" && !strings.EqualFold(e.Topic, req.Topic) {
			continue
		}
		if req.Keyword != "" && !strings.Contains(strings.ToLower(e.Content), strings.ToLower(req.Keyword)) {
			continue
		}
		rsp.Entries = append(rsp.Entries, e)
	}
	return nil
}

// ---------------------------------------------------------------------------
// main
// ---------------------------------------------------------------------------

func main() {
	// Ollama Cloud is the default. Override with env vars for local Ollama.
	baseURL := os.Getenv("OLLAMA_BASE_URL")
	if baseURL == "" {
		baseURL = "https://ollama.com/v1"
	}
	model := os.Getenv("OLLAMA_MODEL")
	if model == "" {
		model = "gemma4:31b-cloud"
	}
	apiKey := os.Getenv("OLLAMA_API_KEY")

	fmt.Println("╔══════════════════════════════════════════╗")
	fmt.Println("║       Ollama-Powered Go Micro Agent      ║")
	fmt.Println("╚══════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("  Ollama URL:  %s\n", baseURL)
	fmt.Printf("  Model:       %s\n", model)
	if apiKey != "" {
		fmt.Printf("  API Key:     (set)\n")
	} else {
		fmt.Printf("  API Key:     (none — set OLLAMA_API_KEY)\n")
	}
	fmt.Println()

	// 1. Start the knowledge service. Its handlers become agent tools.
	svc := micro.NewService("knowledge")
	svc.Handle(new(KnowledgeService))
	go svc.Run()

	// Give the service a moment to register.
	time.Sleep(2 * time.Second)

	// 2. Create the agent. It discovers the knowledge service endpoints
	//    as tools automatically, plus gets a custom "current_time" tool.
	ag := micro.NewAgent("ollama-assistant",
		micro.AgentServices("knowledge"),
		micro.AgentPrompt(
			"You are a helpful knowledge assistant. You can search and add to "+
				"a knowledge base using the knowledge service tools. "+
				"When asked about the current time, use the current_time tool. "+
				"Be concise and factual.",
		),
		micro.AgentProvider("ollama"),
		micro.AgentModel(model),
		micro.AgentAPIKey(apiKey),
		micro.AgentBaseURL(baseURL),
		micro.AgentMaxSteps(10),
		micro.AgentLoopLimit(3),
		// Custom tool — any function, not tied to a service.
		agent.WithTool(
			"current_time",
			"Get the current date and time in a human-readable format",
			map[string]any{
				"timezone": map[string]any{
					"type":        "string",
					"description": "Optional timezone (defaults to local)",
				},
			},
			func(ctx context.Context, input map[string]any) (string, error) {
				tz, _ := input["timezone"].(string)
				if tz == "" {
					return time.Now().Format("2006-01-02 15:04:05 MST"), nil
				}
				loc, err := time.LoadLocation(tz)
				if err != nil {
					return "", fmt.Errorf("unknown timezone: %s", tz)
				}
				return time.Now().In(loc).Format("2006-01-02 15:04:05 MST"), nil
			},
		),
	)

	// 3. Seed initial knowledge via the agent's first question.
	questions := []string{
		"What time is it now?",
		"Add a new knowledge entry: topic 'go', content 'Go interfaces are implicit — a type implements an interface by having the required methods.'",
		"Add another entry: topic 'go', content 'Go is a statically typed, compiled language designed at Google.'",
		"Add another entry: topic 'ai', content 'Large language models generate text by predicting the next token in a sequence.'",
		"Search the knowledge base for entries about Go.",
		"Search for everything in the knowledge base.",
	}

	fmt.Println("─── Agent Demo ───")
	fmt.Println()

	for i, q := range questions {
		fmt.Printf("Q%d: %s\n", i+1, q)
		fmt.Print("A: ")

		resp, err := ag.Ask(context.Background(), q)
		if err != nil {
			fmt.Printf("error: %v\n", err)
			fmt.Println()
			continue
		}

		// Show tool calls the agent made.
		if len(resp.ToolCalls) > 0 {
			for _, tc := range resp.ToolCalls {
				args, _ := json.Marshal(tc.Input)
				fmt.Printf("  [tool] %s(%s)\n", tc.Name, string(args))
			}
		}

		fmt.Println(resp.Reply)
		if resp.Reply == "" && len(resp.ToolCalls) == 0 {
			fmt.Println("(no response)")
		}
		fmt.Println()
	}

	// 4. Streaming demonstration.
	fmt.Println("─── Streaming Demo ───")
	fmt.Println()
	streamQ := "Explain what Go Micro is in two sentences."
	fmt.Printf("Q: %s\n", streamQ)
	fmt.Print("A: ")

	stream, err := ag.Stream(context.Background(), streamQ)
	if err != nil {
		fmt.Printf("stream error: %v\n", err)
	} else {
		for {
			chunk, err := stream.Recv()
			if err != nil {
				break
			}
			if chunk.Reply != "" {
				fmt.Print(chunk.Reply)
			}
		}
		fmt.Println()
	}

	fmt.Println()
	fmt.Println("Done.")
}
