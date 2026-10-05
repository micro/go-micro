package main

import (
	"embed"
	"go-micro.dev/v6/cmd"
	"os/exec"
	"runtime/debug"
	"strings"

	// Link every plugin so CLI flag selection (--registry etcd, --broker nats,
	// --profile nats, ...) keeps working; library users omit this import.
	_ "go-micro.dev/v6/cmd/defaults"

	_ "go-micro.dev/v6/cmd/micro/a2a"
	_ "go-micro.dev/v6/cmd/micro/ai"
	_ "go-micro.dev/v6/cmd/micro/api"
	_ "go-micro.dev/v6/cmd/micro/chat"
	_ "go-micro.dev/v6/cmd/micro/cli"
	_ "go-micro.dev/v6/cmd/micro/cli/build"
	_ "go-micro.dev/v6/cmd/micro/cli/deploy"
	_ "go-micro.dev/v6/cmd/micro/flow"
	"go-micro.dev/v6/cmd/micro/gateway"
	_ "go-micro.dev/v6/cmd/micro/inspect"
	_ "go-micro.dev/v6/cmd/micro/loop"
	_ "go-micro.dev/v6/cmd/micro/mcp"
	_ "go-micro.dev/v6/cmd/micro/resource"
	_ "go-micro.dev/v6/cmd/micro/run"
)

//go:embed web/styles.css web/main.js web/templates/*
var webFS embed.FS

var version = "v6-dev"

// getVersion reports the ldflags-injected release version when set,
// else the module or VCS revision stamped by the go tool, so binaries
// installed via `go install ./cmd/micro` report the git commit instead
// of the v6-dev fallback. Same build-info pattern as microVersion
// in cmd/micro/cli/new (ponytail: keep in sync, don't abstract).
func getVersion() string {
	if version != "v6-dev" && version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return info.Main.Version
		}
		var revision string
		var modified bool
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				revision = setting.Value
			case "vcs.modified":
				modified = setting.Value == "true"
			}
		}
		if revision != "" {
			if len(revision) > 7 {
				revision = revision[:7]
			}
			if modified {
				revision += "-dirty"
			}
			return revision
		}
	}
	// The go tool stamps no VCS info for linked worktrees (.git is a file,
	// not a dir), so dev installs from a worktree get here. Ask git about
	// the repo in the current directory instead; silent when absent.
	// ponytail: one exec, ~15ms, dev-builds only; releases return above.
	if out, err := exec.Command("git", "describe", "--tags", "--always", "--dirty").Output(); err == nil {
		if rev := strings.TrimSpace(string(out)); rev != "" {
			return rev
		}
	}
	return version
}

func init() {
	gateway.HTML = webFS
}

func main() {
	_ = cmd.Init(
		cmd.Name("micro"),
		cmd.Version(getVersion()),
	)
}
