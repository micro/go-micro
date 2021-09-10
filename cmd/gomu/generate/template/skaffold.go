package template

// SkaffoldCFG is the Skaffold config template used for new projects.
var SkaffoldCFG = `---

apiVersion: skaffold/v2beta21
kind: Config
metadata:
  name: {{.Dir}}
build:
  artifacts:
  - image: {{.Dir}}
deploy:
  kubectl:
    manifests:
    - resources/*.yaml
`
