# HTTP Server

The HTTP Server is a go-micro.Server. It's a partial implementation which strips out codecs, transports, etc but enables you 
to create a HTTP Server that could potentially be used for REST based API services.

## Usage

```go
import (
	"net/http"

	"github.com/micro/go-micro/server"
	httpServer "github.com/asim/go-micro/plugins/server/http"
)

func main() {
	srv := httpServer.NewServer(
		server.Name("helloworld"),
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`hello world`))
	})

	hd := srv.NewHandler(mux)

	srv.Handle(hd)
	srv.Start()
	srv.Register()
}
```

Or as part of a service

```go
import (
	"net/http"

	"github.com/asim/go-micro/v3"
	"github.com/micro/go-micro/server"
	httpServer "github.com/asim/go-micro/plugins/server/http"
)

func main() {
	srv := httpServer.NewServer(
		server.Name("helloworld"),
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`hello world`))
	})

	hd := srv.NewHandler(mux)

	srv.Handle(hd)

	service := micro.NewService(
		micro.Server(srv),
	)
	service.Init()
	service.Run()
}
```
