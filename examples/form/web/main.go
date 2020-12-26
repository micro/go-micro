package main

import (
	"fmt"
	"net/http"

	"github.com/micro/go-micro/v2/web"
)

func index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, `<html>
<body>
<h1>This is a regular form</h1>
<form action="http://localhost:8080/form/submit" method="POST">
<input type="text" id="thing" name="thing" />
<button>submit</button>
</form>
<h1>This is a multipart form</h1>
<form action="http://localhost:8080/form/multipart" method="POST" enctype="multipart/form-data">
<input type="text" id="thing" name="thing" />
<button>submit</button>
</form>
</body>
</html>
`)
}

func main() {
	service := web.NewService(
		web.Name("go.micro.web.form"),
	)
	service.Init()
	service.HandleFunc("/", index)
	service.Run()
}
