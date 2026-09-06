// Package inspect registers the 'micro inspect' CLI command.
package inspect

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/urfave/cli/v2"
	goagent "go-micro.dev/v6/agent"
	"go-micro.dev/v6/cmd"
	aiflow "go-micro.dev/v6/flow"
	"go-micro.dev/v6/store"
)

func init() {
	cmd.Register(&cli.Command{
		Name:  "inspect",
		Usage: "Inspect recent agent and workflow activity",
		Description: `Inspect is the CLI checkpoint in the local scaffold → run → chat → inspect loop.
It reads durable local run history, so it works after the agent or flow has stopped.`,
		Subcommands: []*cli.Command{
			{
				Name:      "agent",
				Usage:     "Show recent recorded runs for an agent",
				ArgsUsage: "[agent]",
				Flags:     inspectAgentFlags(),
				Action:    inspectAgent,
			},
			{
				Name:      "flow",
				Usage:     "Show durable run history for a flow",
				ArgsUsage: "[flow]",
				Flags:     inspectFlowFlags(),
				Action:    inspectFlow,
			},
		},
	})
}

func inspectAgentFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{Name: "json", Usage: "Print run data as JSON for automation"},
		&cli.StringFlag{Name: "run", Usage: "Show the complete versioned record for this run id"},
		&cli.StringFlag{Name: "status", Usage: "Only show runs with this status (running, done, canceled, timeout, rate_limited, auth, configuration, unavailable, provider_error, error, refused)"},
		&cli.StringFlag{Name: "trace", Usage: "Only show runs whose trace id matches this full id or prefix"},
		&cli.IntFlag{Name: "limit", Usage: "Show the most recently updated N runs"},
	}
}

func inspectFlowFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{Name: "json", Usage: "Print durable run history as JSON for automation"},
		&cli.StringFlag{Name: "run", Usage: "Show the complete versioned record for this run id"},
		&cli.BoolFlag{Name: "pending", Usage: "Only show runs that have not completed"},
		&cli.StringFlag{Name: "status", Usage: "Only show runs with this status (running, done, failed)"},
		&cli.IntFlag{Name: "limit", Usage: "Show the most recently updated N runs"},
		&cli.StringFlag{Name: "stage", Usage: "Only show runs currently checkpointed at this stage"},
	}
}

func inspectAgent(c *cli.Context) error {
	name := c.Args().First()
	if name == "" {
		return fmt.Errorf("agent name required: micro inspect agent <name>")
	}
	if runID := c.String("run"); runID != "" {
		record, err := goagent.LoadRunRecord(store.DefaultStore, name, runID)
		if err != nil {
			return err
		}
		return writeAgentRunRecord(os.Stdout, record, c.Bool("json"))
	}
	opts := goagent.RunListOptions{Status: c.String("status"), TraceID: c.String("trace"), Limit: c.Int("limit")}
	runs, err := goagent.ListRunSummariesWithOptions(store.DefaultStore, name, opts)
	if err != nil {
		return err
	}
	return writeAgentInspection(os.Stdout, name, runs, c.Bool("json"))
}

func writeAgentRunRecord(w io.Writer, record goagent.RunRecord, asJSON bool) error {
	if asJSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(record)
	}
	if len(record.Events) == 0 {
		fmt.Fprintf(w, "  No recorded events for agent %q run %q.\n", record.Summary.Agent, record.Summary.RunID)
		return nil
	}
	summary := record.Summary
	fmt.Fprintf(w, "  Agent %q run %q  schema=%d\n", summary.Agent, summary.RunID, record.SchemaVersion)
	fmt.Fprintf(w, "  status=%s  events=%d  started=%s  updated=%s  duration_ms=%d",
		summary.Status, summary.Events, summary.StartedAt.Format(time.RFC3339Nano), summary.UpdatedAt.Format(time.RFC3339Nano), summary.DurationMS)
	if summary.ParentID != "" {
		fmt.Fprintf(w, "  parent=%s", summary.ParentID)
	}
	if summary.TraceID != "" {
		fmt.Fprintf(w, "  trace=%s", summary.TraceID)
	}
	if summary.SpanID != "" {
		fmt.Fprintf(w, "  span=%s", summary.SpanID)
	}
	if summary.Checkpoint != "" {
		fmt.Fprintf(w, "  checkpoint=%s", summary.Checkpoint)
	}
	if summary.Stage != "" {
		fmt.Fprintf(w, "  stage=%s", summary.Stage)
	}
	if summary.LastErrorKind != "" {
		fmt.Fprintf(w, "  error_kind=%s", summary.LastErrorKind)
	}
	if summary.Spent > 0 {
		fmt.Fprintf(w, "  spent=%d", summary.Spent)
	}
	if summary.LastError != "" {
		fmt.Fprintf(w, "  error=%q", summary.LastError)
	}
	fmt.Fprintln(w)
	if summary.Flow != "" {
		fmt.Fprintf(w, "  Origin  flow=%s", summary.Flow)
		if summary.Step != "" {
			fmt.Fprintf(w, "  step=%s", summary.Step)
		}
		if summary.Dispatch != "" {
			fmt.Fprintf(w, "  dispatch=%s", summary.Dispatch)
		}
		if summary.Trigger != "" {
			fmt.Fprintf(w, "  trigger=%q", summary.Trigger)
		}
		fmt.Fprintln(w)
		if summary.ParentID != "" {
			fmt.Fprintf(w, "    parent: micro inspect flow %s --run %s\n", summary.Flow, summary.ParentID)
		}
	}
	fmt.Fprintln(w, "  Timeline")
	for _, event := range record.Events {
		fmt.Fprintf(w, "  %s  kind=%s", event.Time.Format(time.RFC3339Nano), event.Kind)
		if event.Name != "" {
			fmt.Fprintf(w, "  name=%q", event.Name)
		}
		if event.Provider != "" || event.Model != "" {
			fmt.Fprintf(w, "  provider=%s  model=%s", event.Provider, event.Model)
		}
		if event.Attempt > 0 || event.MaxAttempts > 0 {
			fmt.Fprintf(w, "  attempt=%d/%d", event.Attempt, event.MaxAttempts)
		}
		if event.LatencyMS > 0 {
			fmt.Fprintf(w, "  latency_ms=%d", event.LatencyMS)
		}
		if event.Tokens.TotalTokens > 0 {
			fmt.Fprintf(w, "  tokens=%d/%d/%d", event.Tokens.InputTokens, event.Tokens.OutputTokens, event.Tokens.TotalTokens)
		}
		if event.Refused != "" {
			fmt.Fprintf(w, "  refused=%q", event.Refused)
		}
		if event.Status != "" {
			fmt.Fprintf(w, "  status=%s", event.Status)
		}
		if event.ErrorKind != "" {
			fmt.Fprintf(w, "  error_kind=%s", event.ErrorKind)
		}
		if event.Error != "" {
			fmt.Fprintf(w, "  error=%q", event.Error)
		}
		if event.InputChars > 0 {
			fmt.Fprintf(w, "  input_chars=%d", event.InputChars)
		}
		if event.Spent > 0 || event.ToolSpend > 0 {
			fmt.Fprintf(w, "  spent=%d  tool_spend=%d", event.Spent, event.ToolSpend)
		}
		fmt.Fprintln(w)
	}
	return nil
}

func writeAgentInspection(w io.Writer, name string, runs []goagent.RunSummary, asJSON bool) error {
	if asJSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(runs)
	}
	if len(runs) == 0 {
		fmt.Fprintf(w, "  No agent runs recorded for %q. After chatting, try: micro inspect agent %s\n", name, name)
		return nil
	}
	fmt.Fprintf(w, "  Agent %q runs\n", name)
	for _, run := range runs {
		fmt.Fprintf(w, "  %s  status=%s  events=%d  last=%s", run.RunID, run.Status, run.Events, run.LastKind)
		if run.Checkpoint != "" {
			fmt.Fprintf(w, "  checkpoint=%s", run.Checkpoint)
		}
		if run.Stage != "" {
			fmt.Fprintf(w, "  stage=%s", run.Stage)
		}
		if run.LastErrorKind != "" {
			fmt.Fprintf(w, "  error_kind=%s", run.LastErrorKind)
		}
		if run.Spent > 0 {
			fmt.Fprintf(w, "  spent=%d", run.Spent)
		}
		if run.LastError != "" {
			fmt.Fprintf(w, "  error=%q", run.LastError)
		}
		if run.TraceID != "" {
			fmt.Fprintf(w, "  trace=%s", shortID(run.TraceID))
		}
		if run.Flow != "" {
			fmt.Fprintf(w, "  flow=%s", run.Flow)
		}
		if run.Step != "" {
			fmt.Fprintf(w, "  step=%s", run.Step)
		}
		fmt.Fprintln(w)
		if run.Flow != "" && run.ParentID != "" {
			fmt.Fprintf(w, "    parent:  micro inspect flow %s --run %s\n", run.Flow, run.ParentID)
		}
		writeAgentRunBreadcrumbs(w, name, run)
	}
	return nil
}

func writeAgentRunBreadcrumbs(w io.Writer, name string, run goagent.RunSummary) {
	if run.Stage == "input-required" {
		fmt.Fprintf(w, "    inspect: micro agent history %s %s\n", name, run.RunID)
		fmt.Fprintf(w, "    input:   micro agent resume-input %s %s --input <text>\n", name, run.RunID)
		return
	}
	if !isResumableAgentRun(run) {
		return
	}
	fmt.Fprintf(w, "    inspect: micro agent history %s %s\n", name, run.RunID)
	fmt.Fprintf(w, "    resume:  call micro.AgentResume(ctx, agent, %q) after recreating the agent with the same checkpoint store\n", run.RunID)
	fmt.Fprintf(w, "    stream:  call micro.ResumeStreamAsk(ctx, agent, %q) to resume with streaming events\n", run.RunID)
}

func isResumableAgentRun(run goagent.RunSummary) bool {
	switch run.Status {
	case "running", "error", "failed", "refused":
		return run.Checkpoint != "done" || run.Stage != ""
	default:
		return false
	}
}

func inspectFlow(c *cli.Context) error {
	name := c.Args().First()
	if name == "" {
		return fmt.Errorf("flow name required: micro inspect flow <name>")
	}
	checkpoint := aiflow.StoreCheckpoint(nil, name)
	if runID := c.String("run"); runID != "" {
		record, err := aiflow.LoadRunRecord(context.Background(), checkpoint, name, runID)
		if err != nil {
			return err
		}
		return writeFlowRunRecord(os.Stdout, record, c.Bool("json"))
	}
	runs, err := checkpoint.List(context.Background())
	if err != nil {
		return err
	}
	runs = filterFlowInspection(runs, c.Bool("pending"), c.String("status"), c.String("stage"), c.Int("limit"))
	return writeFlowInspection(os.Stdout, name, runs, c.Bool("json"), c.Bool("pending"))
}

func writeFlowRunRecord(w io.Writer, record aiflow.RunRecord, asJSON bool) error {
	if asJSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(record)
	}
	run := record.Run
	if run.Status == "" {
		fmt.Fprintf(w, "  No recorded flow %q run %q.\n", run.Flow, run.ID)
		return nil
	}
	fmt.Fprintf(w, "  Flow %q run %q  schema=%d\n", run.Flow, run.ID, record.SchemaVersion)
	fmt.Fprintf(w, "  status=%s  started=%s  updated=%s", run.Status, run.Started.Format(time.RFC3339Nano), run.Updated.Format(time.RFC3339Nano))
	if run.ParentID != "" {
		fmt.Fprintf(w, "  parent=%s", run.ParentID)
	}
	if run.Dispatch != "" {
		fmt.Fprintf(w, "  dispatch=%s", run.Dispatch)
	}
	if run.Trigger != "" {
		fmt.Fprintf(w, "  trigger=%q", run.Trigger)
	}
	if run.State.Stage != "" {
		fmt.Fprintf(w, "  stage=%s", run.State.Stage)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Steps")
	for _, step := range run.Steps {
		fmt.Fprintf(w, "  %s  status=%s  attempts=%d", step.Name, step.Status, step.Attempts)
		if step.Service != "" {
			fmt.Fprintf(w, "  service=%s  endpoint=%s", step.Service, step.Endpoint)
		}
		if step.Agent != "" {
			fmt.Fprintf(w, "  agent=%s", step.Agent)
		}
		if step.ChildRunID != "" {
			fmt.Fprintf(w, "  child_run=%s", step.ChildRunID)
		}
		if step.VerificationStatus != "" {
			fmt.Fprintf(w, "  verification=%s", step.VerificationStatus)
		}
		if step.ErrorKind != "" {
			fmt.Fprintf(w, "  error_kind=%s", step.ErrorKind)
		}
		if step.Error != "" {
			fmt.Fprintf(w, "  error=%q", step.Error)
		}
		fmt.Fprintln(w)
		if step.Agent != "" && step.ChildRunID != "" {
			fmt.Fprintf(w, "    inspect: micro inspect agent %s --run %s\n", step.Agent, step.ChildRunID)
		}
	}
	return nil
}

func filterFlowInspection(runs []aiflow.Run, pending bool, status, stage string, limit int) []aiflow.Run {
	filtered := make([]aiflow.Run, 0, len(runs))
	for _, run := range runs {
		if pending && run.Status == "done" {
			continue
		}
		if status != "" && run.Status != status {
			continue
		}
		if stage != "" && run.State.Stage != stage {
			continue
		}
		filtered = append(filtered, run)
	}
	if limit > 0 && len(filtered) > limit {
		return filtered[len(filtered)-limit:]
	}
	return filtered
}

func writeFlowInspection(w io.Writer, name string, runs []aiflow.Run, asJSON, pending bool) error {
	if asJSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(runs)
	}
	if len(runs) == 0 {
		if pending {
			fmt.Fprintf(w, "  No pending flow runs recorded for %q.\n", name)
			return nil
		}
		fmt.Fprintf(w, "  No flow runs recorded for %q. After executing a durable flow, try: micro inspect flow %s\n", name, name)
		return nil
	}
	fmt.Fprintf(w, "  Flow %q runs\n", name)
	for _, run := range runs {
		stage := run.State.Stage
		if stage == "" {
			stage = "-"
		}
		fmt.Fprintf(w, "  %s  status=%s  stage=%s  steps=%d", shortID(run.ID), run.Status, stage, len(run.Steps))
		for _, step := range run.Steps {
			if step.Error != "" {
				fmt.Fprintf(w, "  error=%q", step.Error)
				break
			}
		}
		fmt.Fprintln(w)
		for _, step := range run.Steps {
			if step.Agent != "" && step.ChildRunID != "" {
				fmt.Fprintf(w, "    child: micro inspect agent %s --run %s\n", step.Agent, step.ChildRunID)
			}
		}
	}
	return nil
}

func shortID(id string) string {
	if len(id) <= 12 {
		return id
	}
	return id[:12]
}
