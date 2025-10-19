module go-micro.dev/v5/cmd/protoc-gen-micro

go 1.24

require (
    google.golang.org/protobuf v1.36.6
)

// Use the root module for other dependencies during development. Keep this file
// minimal to ensure `go install go-micro.dev/v5/cmd/protoc-gen-micro@latest`
// resolves to the correct module path for future v5 tagged releases.
