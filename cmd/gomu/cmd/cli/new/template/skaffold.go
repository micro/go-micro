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

// SkaffoldDEP is the Kubernetes deployment manifest template used for new
// projects.
var SkaffoldDEP = `---

apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.Alias}}
  labels:
    app: {{.Alias}}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: {{.Alias}}
  template:
    metadata:
      labels:
        app: {{.Alias}}
    spec:
      containers:
      - name: {{.Alias}}
        image: {{.Alias}}:latest
`
