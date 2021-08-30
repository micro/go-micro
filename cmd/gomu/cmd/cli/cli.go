package cli

import (
	"sort"

	"github.com/asim/go-micro/cmd/gomu/cmd"
	"github.com/asim/go-micro/cmd/gomu/cmd/cli/call"
	"github.com/asim/go-micro/cmd/gomu/cmd/cli/describe"
	"github.com/asim/go-micro/cmd/gomu/cmd/cli/new"
	"github.com/asim/go-micro/cmd/gomu/cmd/cli/run"
	"github.com/asim/go-micro/cmd/gomu/cmd/cli/services"
	"github.com/asim/go-micro/cmd/gomu/cmd/cli/stream"
	mcli "github.com/micro/cli/v2"
)

func init() {
	cmd.Register(
		call.NewCommand(),
		describe.NewCommand(),
		new.NewCommand(),
		run.NewCommand(),
		services.NewCommand(),
		stream.NewCommand(),
	)

	sort.Sort(mcli.FlagsByName(cmd.App().Flags))
	sort.Sort(mcli.CommandsByName(cmd.App().Commands))
}
