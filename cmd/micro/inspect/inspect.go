// Package inspect registers the 'micro inspect' CLI command.
package inspect

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

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
		&cli.BoolFlag{Name: "json", Usage: "Print run summaries as JSON for automation"},
		&cli.StringFlag{Name: "status", Usage: "Only show runs with this status (running, done, canceled, timeout, rate_limited, auth, configuration, unavailable, provider_error, error, refused)"},
		&cli.StringFlag{Name: "trace", Usage: "Only show runs whose trace id matches this full id or prefix"},
		&cli.IntFlag{Name: "limit", Usage: "Show the most recently updated N runs"},
	}
}

func inspectFlowFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{Name: "json", Usage: "Print durable run history as JSON for automation"},
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
	opts := goagent.RunListOptions{Status: c.String("status"), TraceID: c.String("trace"), Limit: c.Int("limit")}
	runs, err := goagent.ListRunSummariesWithOptions(store.DefaultStore, name, opts)
	if err != nil {
		return err
	}
	return writeAgentInspection(os.Stdout, name, runs, c.Bool("json"))
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

func inspectFlow(c *cli.Context) error {
	name := c.Args().First()
	if name == "" {
		return fmt.Errorf("flow name required: micro inspect flow <name>")
	}
	runs, err := aiflow.StoreCheckpoint(nil, name).List(context.Background())
	if err != nil {
		return err
	}
	runs = filterFlowInspection(runs, c.Bool("pending"), c.String("status"), c.String("stage"), c.Int("limit"))
	return writeFlowInspection(os.Stdout, name, runs, c.Bool("json"), c.Bool("pending"))
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
	}
	return nil
}

func shortID(id string) string {
	if len(id) <= 12 {
		return id
	}
	return id[:12]
}
