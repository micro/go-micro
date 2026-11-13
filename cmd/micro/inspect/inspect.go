// Package inspect registers the 'micro inspect' CLI command.
package inspect

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/urfave/cli/v2"
	goagent "go-micro.dev/v6/agent"
	"go-micro.dev/v6/cmd"
	"go-micro.dev/v6/store"
)

func init() {
	cmd.Register(&cli.Command{
		Name:  "inspect",
		Usage: "Inspect recent agent activity",
		Description: `Inspect is the CLI checkpoint in the local scaffold → run → chat → inspect loop.
It reads durable local run history, so it works after the agent has stopped.`,
		Subcommands: []*cli.Command{
			{
				Name:      "agent",
				Usage:     "Show recent recorded runs for an agent",
				ArgsUsage: "[agent]",
				Flags:     inspectAgentFlags(),
				Action:    inspectAgent,
			},
		},
	})
}

func inspectAgentFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{Name: "json", Usage: "Print run data as JSON for automation"},
		&cli.StringFlag{Name: "run", Usage: "Show the complete versioned record for this run id"},
		&cli.StringFlag{Name: "session", Usage: "Filter by conversation ID"},
		&cli.StringFlag{Name: "status", Usage: "Only show runs with this status (running, done, canceled, timeout, rate_limited, auth, configuration, unavailable, provider_error, error, refused)"},
		&cli.StringFlag{Name: "trace", Usage: "Only show runs whose trace id matches this full id or prefix"},
		&cli.IntFlag{Name: "limit", Usage: "Show the most recently updated N runs"},
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
	opts := goagent.RunListOptions{Session: c.String("session"), Status: c.String("status"), TraceID: c.String("trace"), Limit: c.Int("limit")}
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
		if run.Session != "" {
			fmt.Fprintf(w, "  session=%s", run.Session)
		}
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

func shortID(id string) string {
	if len(id) <= 12 {
		return id
	}
	return id[:12]
}
