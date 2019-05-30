# Log

This is the global logger for all micro based libraries which makes use of [github.com/go-log/log](https://github.com/go-log/log). 

It defaults the logger to the stdlib log implementation. 

## Set Logger

Set the logger for micro libraries

```go
// import go-micro/util/log
import "github.com/micro/go-micro/util/log"

// SetLogger expects github.com/go-log/log.Logger interface
log.SetLogger(mylogger)
```
