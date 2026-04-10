# Graceful Stop Demo

This example demonstrates the intended shutdown behavior after the gRPC graceful-stop patch.

## Run

```bash
go run ./examples/graceful-stop
```

## Expected behavior

- one long-running RPC starts
- shutdown begins while that RPC is still running
- new RPCs stop being accepted shortly after shutdown starts
- the in-flight RPC is allowed to finish

Typical output:

```text
long RPC is running; starting shutdown
new RPC rejected after shutdown began: ...
long RPC completed: slept for 1500ms
done
```

There may be a small race window where the first post-stop RPC is still accepted once before subsequent new RPCs are rejected. The important part is that in-flight RPCs are drained while new RPCs are cut off.

## Automated check

```bash
go test ./server/grpc -run TestGracefulStopRejectsNewRPCsButAllowsInFlightRPCs -v
```

## Environment

- no special environment variables are required
- the demo may print a TLS warning from `go-micro`; it is unrelated to this change
