# Redis Broker

This is a basic implementation for a Redis broker that relies on the native [pub/sub](http://redis.io/topics/pubsub) feature.

Note that broker protocol supports two features that Redis does not support. Subscribers of messages cannot acknowledge back to the Redis server that they received the message and was successfully processed. Thus, if an errors occurs the message will be lost.

The second limitation is that the Redis broker does not support the queue abstraction defined on the broker for distributing messages across subscribers that are apart of the same queue. This is because Redis is not a dedicated broker, but the pub/sub feature is simply a feature of the overall system.

Note that queues can be implemented in Redis, so this feature could theoretically be supported.
