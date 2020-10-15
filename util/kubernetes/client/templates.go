package client

var templates = map[string]string{
	"deployment":      deploymentTmpl,
	"service":         serviceTmpl,
	"namespace":       namespaceTmpl,
	"secret":          secretTmpl,
	"serviceaccount":  serviceAccountTmpl,
	"networkpolicies": networkPolicyTmpl,
	"networkpolicy":   networkPolicyTmpl,
	"resourcequota":   resourceQuotaTmpl,
}

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
      serviceAccountName: {{ .Spec.Template.PodSpec.ServiceAccountName }}
      containers:
      {{- with .Spec.Template.PodSpec.Containers }}
      {{- range . }}
        - name: {{ .Name }}
          env:
          {{- with .Env }}
          {{- range . }}
          - name: "{{ .Name }}"
            value: "{{ .Value }}"
          {{- if .ValueFrom }}
          {{- with .ValueFrom }}
            valueFrom: 
              {{- if .SecretKeyRef }}
              {{- with .SecretKeyRef }}
              secretKeyRef:
                key: {{ .Key }}
                name: {{ .Name }}
                optional: {{ .Optional }}
              {{- end }}
              {{- end }}
          {{- end }}
          {{- end }}
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
          imagePullPolicy: IfNotPresent
          ports:
          {{- with .Ports }}
          {{- range . }}
          - containerPort: {{ .ContainerPort }}
            name: {{ .Name }}
          {{- end }}
          {{- end }}
          {{- if .ReadinessProbe }}
          {{- with .ReadinessProbe }}
          readinessProbe:
            {{- with .TCPSocket }}
            tcpSocket:
              {{- if .Host }}
              host: {{ .Host }}
              {{- end }}
              port: {{ .Port }}
            {{- end }}
            initialDelaySeconds: {{ .InitialDelaySeconds }}
            periodSeconds: {{ .PeriodSeconds }}
          {{- end }}
          {{- end }}
          {{- if .Resources }}
          {{- with .Resources }}
          resources:
            {{- if .Limits }}
            {{- with .Limits }}
            limits:
              {{- if .Memory }}
              memory: {{ .Memory }}
              {{- end }}
              {{- if .CPU }}
              cpu: {{ .CPU }}
              {{- end }}
              {{- if .EphemeralStorage }}
              ephemeral-storage: {{ .EphemeralStorage }}
              {{- end }}
            {{- end }}
            {{- end }}
            {{- if .Requests }}
            {{- with .Requests }}
            requests:
              {{- if .Memory }}
              memory: {{ .Memory }}
              {{- end }}
              {{- if .CPU }}
              cpu: {{ .CPU }}
              {{- end }}
              {{- if .EphemeralStorage }}
              ephemeral-storage: {{ .EphemeralStorage }}
              {{- end }}
            {{- end }}
            {{- end }}
          {{- end }}
          {{- end }}
          volumeMounts:
          {{- with .VolumeMounts }}
          {{- range . }}
            - name: {{ .Name }}
              mountPath: {{ .MountPath }}
          {{- end }}
          {{- end }}
      {{- end }}
      {{- end }} 
      volumes:
      {{- with .Spec.Template.PodSpec.Volumes }}
      {{- range . }}
        - name: {{ .Name }}
          persistentVolumeClaim:
            claimName: {{ .PersistentVolumeClaim.ClaimName }}
      {{- end }}
      {{- end }}
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

var secretTmpl = `
apiVersion: v1
kind: Secret
type: "{{ .Type }}"
metadata:
  name: "{{ .Metadata.Name }}"
  namespace: "{{ .Metadata.Namespace }}"
  labels:
    {{- with .Metadata.Labels }}
    {{- range $key, $value := . }}
    {{ $key }}: "{{ $value }}"
    {{- end }}
    {{- end }}
data:
  {{- with .Data }}
  {{- range $key, $value := . }}
  {{ $key }}: "{{ $value }}"
  {{- end }}
  {{- end }}
`

var serviceAccountTmpl = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: "{{ .Metadata.Name }}"
  labels:
    {{- with .Metadata.Labels }}
    {{- range $key, $value := . }}
    {{ $key }}: "{{ $value }}"
    {{- end }}
    {{- end }}
imagePullSecrets:
{{- with .ImagePullSecrets }}
{{- range . }}
- name: "{{ .Name }}"
{{- end }}
{{- end }}
`

var networkPolicyTmpl = `
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
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
  podSelector:
    matchLabels:
  ingress:
  - from: # Allow pods in this namespace to talk to each other
    - podSelector: {}
  - from: # Allow pods in the namespaces bearing the specified labels to talk to pods in this namespace:
    - namespaceSelector:
        matchLabels:
          {{- with .AllowedLabels }}
          {{- range $key, $value := . }}
          {{ $key }}: "{{ $value }}"
          {{- end }}
          {{- end }}
`

var resourceQuotaTmpl = `
apiVersion: v1
kind: ResourceQuota
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
  hard:
  {{- if .Limits }}
  {{- with .Limits }}
    {{- if .Memory }}
    limits.memory: {{ .Memory }}
    {{- end }}
    {{- if .CPU }}
    limits.cpu: {{ .CPU }}
    {{- end }}
    {{- if .EphemeralStorage }}
    limits.ephemeral-storage: {{ .EphemeralStorage }}
    {{- end }}
  {{- end }}
  {{- end }}
  {{- if .Requests }}
  {{- with .Requests }}
    {{- if .Memory }}
    requests.memory: {{ .Memory }}
    {{- end }}
    {{- if .CPU }}
    requests.cpu: {{ .CPU }}
    {{- end }}
    {{- if .EphemeralStorage }}
    requests.ephemeral-storage: {{ .EphemeralStorage }}
    {{- end }}
  {{- end }}
  {{- end }}
`
