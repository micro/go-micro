module go-micro.dev/4v

go 1.24

// Phantom module path. The Go proxy cached this as a separate module.
// Every version is retracted so 'go install go-micro.dev/4v@latest' errors or
// resolves to the next non-retracted version instead of this path.
retract [v0.0.0, v1.18.2]
