# URL Source

The URL source reads config from a url.

It uses the `Content-Type` header as the format e.g `application/json` becomes `json`. 
The content itself is not touched. If we can't find a format we'll use the encoder format.

## New Source

Specify url source with url. Defaults to `http://localhost:8080/config`.

```go
urlSource := url.NewSource(
	url.WithURL("http://api.example.com/config"),
)
```

## Load Source

Load the source into config

```go
// Create new config
conf := config.NewConfig()

// Load url source
conf.Load(urlSource)
```

