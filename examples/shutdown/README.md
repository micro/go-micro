# Shutdown

This demonstrates graceful shutdown of a service via context cancellation after 5 seconds

A micro.Service waits on context.Done() or an OS kill signal
