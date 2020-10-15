FROM golang:1.13-alpine

RUN mkdir /user && \
    echo 'nobody:x:65534:65534:nobody:/:' > /user/passwd && \
    echo 'nobody:x:65534:' > /user/group

ENV GO111MODULE=on
RUN apk --no-cache add make git gcc libtool musl-dev ca-certificates dumb-init && \
    rm -rf /var/cache/apk/* /tmp/*

WORKDIR /
COPY ./go.mod ./go.sum ./
RUN go mod download && rm go.mod go.sum
