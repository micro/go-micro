# Zerolog

[Zerolog](https://github.com/rs/zerolog) logger implementation for __go-micro__ [meta logger](https://github.com/micro/go-micro/tree/master/logger).

## Usage

```go
func ExampleWithOut() {
  logger.DefaultLogger = zerolog.NewLogger(logger.WithOutput(os.Stdout), logger.WithLevel(logger.DebugLevel))

  logger.Infof(logger.InfoLevel, "testing: %s", "Infof")

  // Output:
  // {"level":"info","message":"testing: Infof"}
}
```
