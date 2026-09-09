package ai

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/urfave/cli/v2"
	goai "go-micro.dev/v6/ai"
	_ "go-micro.dev/v6/ai/anthropic"
	_ "go-micro.dev/v6/ai/atlascloud"
	_ "go-micro.dev/v6/ai/gemini"
	_ "go-micro.dev/v6/ai/groq"
	_ "go-micro.dev/v6/ai/mistral"
	_ "go-micro.dev/v6/ai/openai"
	_ "go-micro.dev/v6/ai/together"
	"go-micro.dev/v6/cmd"
)

func init() {
	cmd.Register(&cli.Command{
		Name:  "ai",
		Usage: "Inspect AI provider support",
		Subcommands: []*cli.Command{{
			Name:  "providers",
			Usage: "Print the registered AI provider capability matrix",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "json",
					Usage: "Print the capability matrix as JSON",
				},
			},
			Action: providersAction,
		}},
	})
}

func providersAction(c *cli.Context) error {
	rows := goai.CapabilityRows()
	if c.Bool("json") {
		return writeProviderJSON(c.App.Writer, rows)
	}
	writeProviderMatrix(c.App.Writer, rows)
	return nil
}

func writeProviderJSON(w io.Writer, rows []goai.CapabilityRow) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(rows)
}

func writeProviderMatrix(w io.Writer, rows []goai.CapabilityRow) {
	const check = "✓"
	fmt.Fprintln(w, "Provider    Model  Image  Video")
	fmt.Fprintln(w, "--------    -----  -----  -----")
	for _, row := range rows {
		fmt.Fprintf(w, "%-11s %-6s %-6s %-6s\n",
			row.Provider,
			mark(row.Model, check),
			mark(row.Image, check),
			mark(row.Video, check),
		)
	}
}

func mark(ok bool, value string) string {
	if ok {
		return value
	}
	return "-"
}
