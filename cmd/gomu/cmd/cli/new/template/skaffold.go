package template

// SkaffoldCFG is the Skaffold config template used for new projects.
var SkaffoldCFG = `---

apiVersion: skaffold/v2beta21
kind: Config
metadata:
  name: {{.Alias}}
build:
  artifacts:
  - image: {{.Alias}}
deploy:
  kubectl:
    manifests:
    - resources/*.yaml
`
