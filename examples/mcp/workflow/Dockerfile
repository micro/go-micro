FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath -o /service .

FROM scratch
COPY --from=builder /service /service
EXPOSE 9093 3004
CMD ["/service"]
