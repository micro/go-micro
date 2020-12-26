# Redirect

This demonstrates how to http redirect using an API service

## Usage

Run the micro API

```
micro --registry=mdns api
```

Run the redirect API

```
go run main.go --registry=mdns
```

Make request
```
curl -v http://localhost:8080/redirect/url
```

Should return

```
HTTP/1.1 301 Moved Permanently
Location: https://google.com
```
