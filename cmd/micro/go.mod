module go-micro.dev/v5/cmd/micro

go 1.24

require (
    github.com/urfave/cli/v2 v2.27.6
)

// Minimal submodule go.mod for the micro CLI command. This ensures the
// module path matches the import path `go-micro.dev/v5/cmd/micro` for
// go install and tagging. Keep requirements minimal; the root module
// continues to manage transitive dependencies.
module go-micro.dev/v5/cmd/micro

go 1.24

// This is a minimal go.mod for the `cmd/micro` command. Keep dependencies in
// the root module; add explicit requirements here only if needed for CI or
// standalone builds.
