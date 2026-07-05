package agent

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	goagent "go-micro.dev/v6/agent"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/store"
)

type doctorDeps struct {
	getenv       func(string) string
	httpGet      func(string) (*http.Response, error)
	listServices func() ([]*registry.Service, error)
	getService   func(string) ([]*registry.Service, error)
	listRuns     func(string) ([]goagent.RunSummary, error)
}

func defaultDoctorDeps() doctorDeps {
	client := &http.Client{Timeout: 2 * time.Second}
	return doctorDeps{
		getenv:       defaultPreflightDeps().getenv,
		httpGet:      client.Get,
		listServices: registry.ListServices,
		getService:   registry.GetService,
		listRuns: func(name string) ([]goagent.RunSummary, error) {
			return goagent.ListRunSummariesWithOptions(store.DefaultStore, name, goagent.RunListOptions{Limit: 1})
		},
	}
}

func runAgentDoctor(w io.Writer, deps doctorDeps, gateway string) error {
	if gateway == "" {
		gateway = "http://localhost:8080"
	}
	gateway = strings.TrimRight(gateway, "/")
	checks := agentDoctorChecks(deps, gateway)
	failures := 0
	fmt.Fprintln(w, "First-agent recovery doctor")
	for _, check := range checks {
		mark := "✓"
		if !check.OK {
			mark = "✗"
			failures++
		}
		fmt.Fprintf(w, "  %s %s — %s\n", mark, check.Name, check.Detail)
		if !check.OK && check.Fix != "" {
			fmt.Fprintf(w, "    Fix: %s\n", check.Fix)
		}
		if !check.OK && check.Next != "" {
			fmt.Fprintf(w, "    Next: %s\n", check.Next)
		}
	}
	if failures > 0 {
		return fmt.Errorf("first-agent doctor found %d recovery boundary issue(s)", failures)
	}
	fmt.Fprintln(w, "\nReady: gateway, agent registration, chat settings, and inspect history are reachable.")
	return nil
}

func agentDoctorChecks(deps doctorDeps, gateway string) []preflightCheck {
	if deps.getenv == nil {
		deps.getenv = defaultPreflightDeps().getenv
	}
	if deps.httpGet == nil {
		deps.httpGet = http.Get
	}
	if deps.listServices == nil {
		deps.listServices = registry.ListServices
	}
	if deps.getService == nil {
		deps.getService = registry.GetService
	}
	if deps.listRuns == nil {
		deps.listRuns = func(name string) ([]goagent.RunSummary, error) {
			return goagent.ListRunSummariesWithOptions(store.DefaultStore, name, goagent.RunListOptions{Limit: 1})
		}
	}

	checks := []preflightCheck{checkGateway(deps, gateway), checkChatSettings(deps, gateway)}
	agents, regCheck := checkAgentRegistration(deps)
	checks = append(checks, regCheck)
	checks = append(checks, checkRunHistory(deps, agents))
	checks = append(checks, checkProviderConfig(deps))
	return checks
}

func checkGateway(deps doctorDeps, gateway string) preflightCheck {
	resp, err := deps.httpGet(gateway + "/agent")
	if err != nil {
		return preflightCheck{Name: "gateway /agent", Detail: err.Error(), Fix: "Start the local gateway with `micro run`, or pass the matching URL with `micro agent doctor --gateway http://localhost:<port>`.", Next: "Then open " + gateway + "/agent or retry `micro chat`."}
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return preflightCheck{Name: "gateway /agent", Detail: fmt.Sprintf("%s returned %s", gateway+"/agent", resp.Status), Fix: "Confirm `micro run` is serving the web gateway and that auth/proxy settings are not blocking /agent.", Next: "See docs/guides/debugging-agents.html#chat-and-gateway-failures."}
	}
	return preflightCheck{Name: "gateway /agent", OK: true, Detail: gateway + "/agent is reachable"}
}

func checkChatSettings(deps doctorDeps, gateway string) preflightCheck {
	resp, err := deps.httpGet(gateway + "/api/agent/settings")
	if err != nil {
		return preflightCheck{Name: "chat settings endpoint", Detail: err.Error(), Fix: "Keep `micro run` running and retry; the playground uses /api/agent/settings before chat prompts.", Next: "See docs/guides/debugging-agents.html#chat-and-gateway-failures."}
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return preflightCheck{Name: "chat settings endpoint", Detail: fmt.Sprintf("returned %s", resp.Status), Fix: "Check gateway auth/proxy configuration or use the Agent settings page to confirm chat settings load.", Next: "See docs/guides/debugging-agents.html#provider-failures."}
	}
	var settings map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&settings)
	if settings["provider"] != "" || settings["model"] != "" || settings["api_key"] != "" {
		return preflightCheck{Name: "chat settings endpoint", OK: true, Detail: "reachable with saved provider settings"}
	}
	return preflightCheck{Name: "chat settings endpoint", OK: true, Detail: "reachable; no saved provider settings"}
}

func checkAgentRegistration(deps doctorDeps) ([]string, preflightCheck) {
	services, err := deps.listServices()
	if err != nil {
		return nil, preflightCheck{Name: "agent registration", Detail: err.Error(), Fix: "Keep the scaffolded agent process running under `micro run` and retry `micro agent list`.", Next: "See docs/guides/your-first-agent.html#run-your-agent."}
	}
	var agents []string
	for _, svc := range services {
		records, err := deps.getService(svc.Name)
		if err != nil || len(records) == 0 {
			continue
		}
		if serviceIsAgent(records[0]) {
			agents = append(agents, svc.Name)
		}
	}
	if len(agents) == 0 {
		return nil, preflightCheck{Name: "agent registration", Detail: "no registered agent services found", Fix: "Start an agent project with `micro run` and confirm `micro agent list` shows it.", Next: "Use docs/guides/no-secret-first-agent.html for a deterministic no-provider agent."}
	}
	return agents, preflightCheck{Name: "agent registration", OK: true, Detail: "found " + strings.Join(agents, ", ")}
}

func serviceIsAgent(svc *registry.Service) bool {
	if svc.Metadata != nil && svc.Metadata["type"] == "agent" {
		return true
	}
	for _, node := range svc.Nodes {
		if node.Metadata != nil && node.Metadata["type"] == "agent" {
			return true
		}
	}
	return false
}

func checkRunHistory(deps doctorDeps, agents []string) preflightCheck {
	if len(agents) == 0 {
		return preflightCheck{Name: "inspect run history", Detail: "skipped because no agent is registered", Fix: "Fix agent registration first, then chat once and run `micro inspect agent <name>`.", Next: "See docs/guides/debugging-agents.html#inspect-run-history."}
	}
	for _, name := range agents {
		runs, err := deps.listRuns(name)
		if err != nil {
			return preflightCheck{Name: "inspect run history", Detail: err.Error(), Fix: "Ensure the local store is writable and retry `micro inspect agent " + name + "`.", Next: "See docs/guides/debugging-agents.html#inspect-run-history."}
		}
		if len(runs) > 0 {
			return preflightCheck{Name: "inspect run history", OK: true, Detail: "recent runs available for " + name}
		}
	}
	return preflightCheck{Name: "inspect run history", Detail: "no recorded agent runs yet", Fix: "Send one prompt with `micro chat` or the /agent playground, then run `micro inspect agent " + agents[0] + "`.", Next: "See docs/guides/your-first-agent.html#inspect-what-happened."}
}

func checkProviderConfig(deps doctorDeps) preflightCheck {
	check := checkProviderKey(preflightDeps{getenv: deps.getenv})
	check.Name = "provider configuration"
	if !check.OK {
		check.Detail = "no provider key found for live LLM chat"
		check.Fix = "For provider-backed chat, export MICRO_AI_API_KEY or a provider-specific key; for no-secret recovery, use the mock-model walkthrough."
	}
	return check
}
