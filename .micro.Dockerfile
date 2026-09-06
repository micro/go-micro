FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG SVC
RUN CGO_ENABLED=0 go build -o /service ./$SVC

FROM scratch
COPY --from=builder /service /service
EXPOSE 8080
CMD ["/service"]
