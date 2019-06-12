# Sync

Sync is a synchronization library for distributed systems.

## Overview

Distributed systems by their very nature are decoupled and independent. In most cases they must honour 2 out of 3 letters of the CAP theorem 
e.g Availability and Partitional tolerance but sacrificing consistency. In the case of microservices we often offload this concern to 
an external database or eventing system. Go Sync provides a framework for synchronization which can be used in the application by the developer.

## Getting Started

- [Leader](#leader) - leadership election for group coordination
- [Lock](#lock) - distributed locking for exclusive resource access
- [Task](#task) - distributed job execution
- [Time](#time) - provides synchronized time

## Lock

The Lock interface provides distributed locking. Multiple instances attempting to lock the same id will block until available.

```go
import "github.com/micro/go-micro/sync/lock/consul"

lock := consul.NewLock()

// acquire lock
err := lock.Acquire("id")
// handle err

// release lock
err = lock.Release("id")
// handle err
```

## Leader

Leader provides leadership election. Useful where one node needs to coordinate some action.

```go
import (
	"github.com/micro/go-micro/sync/leader"
	"github.com/micro/go-micro/sync/leader/consul"
)

l := consul.NewLeader(
	leader.Group("name"),
)

// elect leader
e, err := l.Elect("id")
// handle err


// operate while leader
revoked := e.Revoked()

for {
	select {
	case <-revoked:
		// re-elect
		e.Elect("id")
	default:
		// leader operation
	}
}

// resign leadership
e.Resign() 
```

## Task

Task provides distributed job execution. It's a simple way to distribute work across a coordinated pool of workers.

```go
import (
	"github.com/micro/go-micro/sync/task"
	"github.com/micro/go-micro/sync/task/local"
)

t := local.NewTask(
	task.WithPool(10),
)

err := t.Run(task.Command{
	Name: "atask",
	Func: func() error {
		// exec some work
		return nil
	},
})

if err != nil {
	// do something
}
```

## Time

Time provides synchronized time. Local machines may have clock skew and time cannot be guaranteed to be the same everywhere. 
Synchronized Time allows you to decide how time is defined for your applications.

```go
import (
	"github.com/micro/go-micro/sync/time/ntp"
)


t := ntp.NewTime()
time, err := t.Now()
```

## TODO

- Event package - strongly consistent event stream e.g kafka
