# Shard Wrapper

The shard wrapper is a way of sharding calls based on metadata key-value so requests can be pinned to a particular backend node.

For example if you want to use session affinity aka sticky sessions you can specify the header value for sessions e.g. X-From-Session or X-From-User

X-From-Session or X-From-User is likely a unique session token or user id. The shard wrapper will look for the key you specify and use the crc32 checksum 
of the value against the nodes much like the gomemcache library. It uses a selector strategy to achieve this.

## Usage

Pass in the wrapper when you create your service with the key you want to shard on

```
wrapper := shard.NewClientWrapper("X-From-Session")

service := micro.NewService(
	micro.Name("foo"),
	micro.WrapClient(wrapper),
)
```

Alternatively wrap the client and use independently

```
wrapper := shard.NewClientWrapper("X-From-Session")

client := wrapper(service.Client())
```

