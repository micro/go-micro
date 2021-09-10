package template

// GitIgnore is the .gitignore template used for new projects.
var GitIgnore = `# don't commit the service binary to vcs
{{.Service}}{{if .Client}}-client{{end}}
`
