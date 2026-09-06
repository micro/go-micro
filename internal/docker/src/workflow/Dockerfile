FROM golang:1.26.0-alpine AS build
RUN apk add --no-cache git
COPY --from=repo . /src
COPY . /service
WORKDIR /service
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod/cache/download \
    go mod download all \
 && CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath -o /out/service .


FROM scratch
COPY --from=build /out/service /usr/local/bin/service
EXPOSE 8085 9085
ENTRYPOINT ["/usr/local/bin/service"]

