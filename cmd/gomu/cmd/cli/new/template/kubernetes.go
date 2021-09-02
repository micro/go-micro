package template

// KubernetesDEP is the Kubernetes deployment manifest template used for new
// projects.
var KubernetesDEP = `---

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
