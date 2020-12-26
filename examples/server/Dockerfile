# Build binary with the following command
# CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w' -o server ./main.go
FROM alpine:3.2
ADD server /
ENTRYPOINT [ "/server" ]
