# Options

Go-micro makes the use of [functional options](https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis). It's a design 
pattern that allows the addition of new options without changing the method signature. 

Each package has an [Option](https://godoc.org/github.com/asim/go-micro#Option) type

```
type Option func(*Options)
```

Options such as the [Name](https://godoc.org/github.com/asim/go-micro#Name) function exist to set a service name

The implementation is as follows

```
func Name(n string) Option {
	return func(o *Options) {
		o.Server.Init(server.Name(n))
	}
}
```

## Usage

Here's an example at the top level

```
import "github.com/asim/go-micro/v3"


service := micro.NewService(
	micro.Name("my.service"),
)
```
