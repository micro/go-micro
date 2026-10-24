---
title: "API Route"
---

# API Route

[API Route](https://www.api-route.com/en/docs/quickstart) provides an OpenAI-compatible gateway. This optional provider uses your own API key and defaults to `gpt-6.1-sol`. Check the authenticated `/v1/models` catalogue for model availability.

```go
import (
    "os"
    "go-micro.dev/v6/model"
    _ "go-micro.dev/v6/model/apiroute"
)

var m = model.New("api-route",
    model.WithAPIKey(os.Getenv("API_ROUTE_API_KEY")),
    model.WithModel("gpt-6.1-sol"),
)
```

For the CLI, export `API_ROUTE_API_KEY` and select `--provider api-route`. The default API root is `https://global.api-route.com`; the provider appends `/v1/chat/completions` or `/v1/models`. A custom `model.WithBaseURL` may include `/v1` and a trailing slash.

Chat, model discovery, tool execution, and plain-text streaming reuse the shared transport. Tool-generation callbacks are buffered until completion so complete assistant metadata can be retained between tool rounds. Tool streaming and image generation are not advertised. Selected requests and conversation content are sent to the configured endpoint.
