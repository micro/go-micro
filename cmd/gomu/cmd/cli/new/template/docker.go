package template

// Dockerfile is the Dockerfile template used for new projects.
var Dockerfile = `FROM golang:alpine AS builder
ENV CGO_ENABLED=0 GOOS=linux
WORKDIR /go/src/{{.Alias}}
RUN apk --update --no-cache add ca-certificates gcc libtool make musl-dev protoc
COPY . /go/src/{{.Alias}}
RUN make {{if not .Client}}init proto {{end}}tidy build

FROM scratch
COPY --from=builder /etc/ssl/certs /etc/ssl/certs
COPY --from=builder /go/src/{{.Alias}}/{{.Alias}} /{{.Alias}}
ENTRYPOINT ["/{{.Alias}}"]
CMD []
`

// DockerIgnore is the .dockerignore template used for new projects.
var DockerIgnore = `.gitignore
Dockerfile{{if .Skaffold}}
resources/
skaffold.yaml{{end}}
`
