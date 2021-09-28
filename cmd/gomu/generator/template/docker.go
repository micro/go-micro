package template

// Dockerfile is the Dockerfile template used for new projects.
var Dockerfile = `FROM golang:alpine AS builder
ENV CGO_ENABLED=0 GOOS=linux
WORKDIR /go/src/{{.Service}}{{if .Client}}-client{{end}}
RUN apk --update --no-cache add ca-certificates gcc libtool make musl-dev protoc
COPY {{if not .Client}}Makefile {{end}}go.mod go.sum ./
RUN {{if not .Client}}make init && {{end}}go mod download
COPY . .
RUN make {{if not .Client}}proto {{end}}tidy build

FROM scratch
COPY --from=builder /etc/ssl/certs /etc/ssl/certs
COPY --from=builder /go/src/{{.Service}}{{if .Client}}-client{{end}}/{{.Service}}{{if .Client}}-client{{end}} /{{.Service}}{{if .Client}}-client{{end}}
ENTRYPOINT ["/{{.Service}}{{if .Client}}-client{{end}}"]
CMD []
`

// DockerIgnore is the .dockerignore template used for new projects.
var DockerIgnore = `.gitignore
Dockerfile{{if .Skaffold}}
resources/
skaffold.yaml{{end}}
`
