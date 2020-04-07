package client

var templates = map[string]string{
	"deployment": deploymentTmpl,
	"service":    serviceTmpl,
	"namespace":  namespaceTmpl,
}

// stripped image pull policy always
// imagePullPolicy: Always
var deploymentTmpl = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: "{{ .Metadata.Name }}"
  namespace: "{{ .Metadata.Namespace }}"
  labels:
    {{- with .Metadata.Labels }}
    {{- range $key, $value := . }}
    {{ $key }}: "{{ $value }}"
    {{- end }}
    {{- end }}
  annotations:
    {{- with .Metadata.Annotations }}
    {{- range $key, $value := . }}
    {{ $key }}: "{{ $value }}"
    {{- end }}
    {{- end }}
spec:
  replicas: {{ .Spec.Replicas }}
  selector:
    matchLabels:
      {{- with .Spec.Selector.MatchLabels }}
      {{- range $key, $value := . }}
      {{ $key }}: "{{ $value }}"
      {{- end }}
      {{- end }}
  template:
    metadata:
      labels:
        {{- with .Spec.Template.Metadata.Labels }}
        {{- range $key, $value := . }}
        {{ $key }}: "{{ $value }}"
        {{- end }}
        {{- end }}
      annotations:
        {{- with .Spec.Template.Metadata.Annotations }}
        {{- range $key, $value := . }}
        {{ $key }}: "{{ $value }}"
        {{- end }}
        {{- end }}
    spec:
      containers:
      {{- with .Spec.Template.PodSpec.Containers }}
      {{- range . }}
        - name: {{ .Name }}
          env:
          {{- with .Env }}
          {{- range . }}
          - name: "{{ .Name }}"
            value: "{{ .Value }}"
          {{- end }}
          {{- end }}
          args:
          {{- range .Args }}
          - {{.}}
          {{- end }}
          command:
          {{- range .Command }}
          - {{.}}
          {{- end }}
          image: {{ .Image }}
          ports:
          {{- with .Ports }}
          {{- range . }}
          - containerPort: {{ .ContainerPort }}
            name: {{ .Name }}
          {{- end}}
          {{- end}}
      {{- end }}
      {{- end}}
`

var serviceTmpl = `
apiVersion: v1
kind: Service
metadata:
  name: "{{ .Metadata.Name }}"
  namespace: "{{ .Metadata.Namespace }}"
  labels:
    {{- with .Metadata.Labels }}
    {{- range $key, $value := . }}
    {{ $key }}: "{{ $value }}"
    {{- end }}
    {{- end }}
spec:
  selector:
    {{- with .Spec.Selector }}
    {{- range $key, $value := . }}
    {{ $key }}: "{{ $value }}"
    {{- end }}
    {{- end }}
  ports:
  {{- with .Spec.Ports }}
  {{- range . }}
  - name: "{{ .Name }}"
    port: {{ .Port }}
    protocol: {{ .Protocol }}
  {{- end }}
  {{- end }}
`

var namespaceTmpl = `
apiVersion: v1
kind: Namespace
metadata:
  name: "{{ .Metadata.Name }}"
  labels:
    {{- with .Metadata.Labels }}
    {{- range $key, $value := . }}
    {{ $key }}: "{{ $value }}"
    {{- end }}
    {{- end }}
`
