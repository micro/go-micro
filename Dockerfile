FROM golang:1.13-alpine
ENV GO111MODULE=on
RUN apk --no-cache add make git gcc libtool musl-dev
WORKDIR /
COPY go.mod .
COPY go.sum .
RUN go mod download && rm go.mod go.sum
