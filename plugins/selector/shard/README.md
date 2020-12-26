# Shard selector

The Shard selector strategy should direct all requests for the given keys to a single node, or if that node is failing, a consistent second-choice etc.

It tries to consistently direct all requests for a given set of sharding keys to a single instance to improve caching memory efficiency.

# Re-balancing requests

When a new node appears, it will get a fair share of requests randomly allocated from across the existing services as the new node will then be higher scoring for approximately `1/count(nodes)` of the ids.

Similarly, when a node disappears, its load will get fairly redistributed amongst the existing remaining nodes.

# Benefits

This benefits us in that memory can be more optimally used by trying to target requests that have support for caching to servers that are more likely to have that data already, whilst not requiring all servers to have all data for everything cached.

Over time, this results in overall memory savings where one is running multiple services, whist still allowing for fractional re-balancing with auto-scaling and unexpected node failure/replacement.

# Usage

This method is a call option, which can be passed into client RPC requests.

## Example

```go
rsp, err := myClient.ClientCall(
    ctx,
    &ClientCallRequest{
    	//...
        SomeID: id,
    },
    shard.Strategy(id),
)
```
