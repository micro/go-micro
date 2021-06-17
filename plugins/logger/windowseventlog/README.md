# WindowsEventLog

[eventlog](https://pkg.go.dev/golang.org/x/sys/windows/svc/eventlog) windows event logger implementation for __go-micro__ [meta logger](https://github.com/micro/go-micro/tree/master/logger).

## Usage

```go
func Example() {
  logger.DefaultLogger = windowseventlog.NewLogger(windowseventlog.WithSrc("test src"), logger.WithEid(1000))

  logger.Infof(logger.InfoLevel, "testing: %s", "Infof")
  
}
```
