FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /micro-mcp-gateway ./cmd/micro-mcp-gateway

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /micro-mcp-gateway /usr/local/bin/micro-mcp-gateway

EXPOSE 3000

ENTRYPOINT ["micro-mcp-gateway"]
CMD ["--address", ":3000"]
