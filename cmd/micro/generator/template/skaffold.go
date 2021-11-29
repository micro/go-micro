package template

// SkaffoldCFG is the Skaffold config template used for new projects.
var SkaffoldCFG = `---

apiVersion: skaffold/v2beta21
kind: Config
metadata:
  name: {{.Service}}{{if .Client}}-client{{end}}
build:
  artifacts:
  - image: {{.Service}}{{if .Client}}-client{{end}}
deploy:
  kubectl:
    manifests:
    - resources/*.yaml
`
