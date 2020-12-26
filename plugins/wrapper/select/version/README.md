# Version Wrapper

The version wrapper is a stateful client wrapper which gives you a ability to select only latest version services. That suitable for easy upgrade running services without downtime.

## Usage

Pass in the wrapper when you create your service

```
wrapper := version.NewClientWrapper()

service := micro.NewService(
	micro.Name("foo"),
	micro.WrapClient(wrapper),
)
```
