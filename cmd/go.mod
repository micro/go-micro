// Phantom module path. This directory is part of the root module
// go-micro.dev/v6; it is NOT a separate module. The Go proxy cached it as one
// because of an old vanity-import / tag layout, which made
// `go install go-micro.dev/v6/cmd/...@latest` resolve here (to v1.18.0, whose
// go.mod declares the obsolete github.com/micro/go-micro path) instead of the
// root module. Every version of this path is retracted so the install falls
// back to go-micro.dev/v6. See issues #2985, #2987.
module go-micro.dev/v6/cmd

go 1.24

retract [v0.0.0, v1.18.2]
