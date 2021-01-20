# logrus

[logrus](https://github.com/sirupsen/logrus) logger implementation for __go-micro__ [meta logger](https://github.com/micro/go-micro/tree/master/logger).

## Usage

```go
import (
	"os"
	"github.com/sirupsen/logrus"
	"github.com/asim/go-micro/v3/logger"
)

func ExampleWithOutput() {
  logger.DefaultLogger = NewLogger(logger.WithOutput(os.Stdout))
  logger.Infof(logger.InfoLevel, "testing: %s", "Infof")
}

func ExampleWithLogger() {
	l:= logrus.New() // *logrus.Logger
	logger.DefaultLogger = NewLogger(WithLogger(l))
  logger.Infof(logger.InfoLevel, "testing: %s", "Infof")
}
```

