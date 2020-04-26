# qson

This is copy from https://github.com/joncalhoun/qson
As author says he is not acrivelly maintains the repo and not plan to do that.

## Usage

You can either turn a URL query param into a JSON byte array, or unmarshal that directly into a Go object.

Transforming the URL query param into a JSON byte array:

```go
import "github.com/joncalhoun/qson"

func main() {
  b, err := qson.ToJSON("bar%5Bone%5D%5Btwo%5D=2&bar[one][red]=112")
  if err != nil {
    panic(err)
  }
  fmt.Println(string(b))
  // Should output: {"bar":{"one":{"red":112,"two":2}}}
}
```

Or unmarshalling directly into a Go object using JSON struct tags:

```go
import "github.com/joncalhoun/qson"

type unmarshalT struct {
	A string     `json:"a"`
	B unmarshalB `json:"b"`
}
type unmarshalB struct {
	C int `json:"c"`
}

func main() {
  var out unmarshalT
  query := "a=xyz&b[c]=456"
  err := Unmarshal(&out, query)
  if err != nil {
  	t.Error(err)
  }
  // out should equal
  //   unmarshalT{
	// 	  A: "xyz",
	// 	  B: unmarshalB{
	// 	  	C: 456,
	// 	  },
	//   }
}
```

To get a query string like in the two previous examples you can use the `RawQuery` field on the [net/url.URL](https://golang.org/pkg/net/url/#URL) type.
