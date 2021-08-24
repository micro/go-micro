# WindowsEventLog

[windows event log](https://pkg.go.dev/golang.org/x/sys/windows/svc/eventlog) implementation for __go-micro__ [meta logger](https://github.com/micro/go-micro/tree/master/logger).

## Usage

Before the first use, it is necessary to initialize the registrar with administrator rights.

The __NewLogger__ function tries to create an event source named __src in the options__ (or by default), but this may not happen, so for proper initialization it is recommended to use the __Init__ function, which returns an error.
```go
func Init() {
  l := windowseventlog.NewLogger(windowseventlog.WithSrc("test src"), logger.WithEid(1000))
  err := l.Init()
  if err != nil {
      //smt
  }
}
```


```go
func Example() {
  logger.DefaultLogger = windowseventlog.NewLogger(windowseventlog.WithSrc("test src"), logger.WithEid(1000))

  logger.Infof(logger.InfoLevel, "testing: %s", "Infof")

}
```
