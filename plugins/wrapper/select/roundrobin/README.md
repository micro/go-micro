# Round Robin Wrapper

The round robin wrapper is a stateful client wrapper which gives you a true round robin strategy for the selector

## Usage

Pass in the wrapper when you create your service

```
wrapper := roundrobin.NewClientWrapper()

service := micro.NewService(
	micro.Name("foo"),
	micro.WrapClient(wrapper),
)
```
