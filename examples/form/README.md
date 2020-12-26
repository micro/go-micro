# Form

This rudimentary example demonstrates how to access a form and multipart form when writing API services

## Contents

- web - is the web front end with the form
- api - is the api service

## Usage

Run the micro api

```
micro api --handler=api
```

Run the micro web

```
micro web
```

Run the api service

``` 
go run api/main.go
```

Run the web service

```
go run web/main.go
```

Browse to localhost:8082/form
