package ai

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v6/cmd"
	gomodel "go-micro.dev/v6/model"
	_ "go-micro.dev/v6/model/anthropic"
	_ "go-micro.dev/v6/model/atlascloud"
	_ "go-micro.dev/v6/model/gemini"
	_ "go-micro.dev/v6/model/groq"
	_ "go-micro.dev/v6/model/mistral"
	_ "go-micro.dev/v6/model/openai"
	_ "go-micro.dev/v6/model/together"
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
	rows := gomodel.CapabilityRows()
	if c.Bool("json") {
		return writeProviderJSON(c.App.Writer, rows)
	}
	writeProviderMatrix(c.App.Writer, rows)
	return nil
}

func writeProviderJSON(w io.Writer, rows []gomodel.CapabilityRow) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(rows)
}

func writeProviderMatrix(w io.Writer, rows []gomodel.CapabilityRow) {
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
