# SQS Broker Plugin for go-micro
Amazon Simple Queue Service broker plugin for `go-micro` allows you to publish and subscribe messages brokered by SQS. This plugin _does not_ (yet) support automatic creation of queues so those will have to exist in your infrastructure before attempting to send/receive.

## AWS Credentials
This plugin uses the official Go SDK for AWS. As such, it will obtain AWS credentials the same way all other `aws-go-sdk` applications do. The plugin explicitly allows the use of the shared credentials file to make development on workstations easier, but you can also supply the usual `AWS_*` environment variables in dev/test/prod environments. Also if you're deploying in EC2/ECS, the `IAM Role` will be picked up automatically and you won't need to supply any credentials.

## Publishing and Subscribing
Publishing and subscribing with the plugin should work just like all other brokers. Simply supply the name of the queue in the publish or subscribe arguments:

```go
broker.Publish("queue.fifo", msg)
...
broker.Subscribe("queue.fifo", subscriberFunc)
```

## Options
If you're using a regular (non-fifo) queue you should be able to get by without having to supply any special options. However, if you need to specify a group identifier for a message or a de-duplication identifier, then you'll have to specify a generator function for those. These are both mandatory when using `FIFO` queues.

### Generator Functions
To specify generator functions for the broker, you add them as options as shown:

```go
cmd.Init()

if err := broker.Init(
    sqs.DeduplicationFunction(dedup),
    sqs.GroupIDFunction(groupid),
); err != nil {
    // handle failure
}
```
These are both functions that accept a pointer to a `broker.Message` as input and return a string in response:

```
func dedup(m *broker.Message) string {

}
func groupid(m *broker.Message) string {

}
```
How you choose to generate group IDs and deduplication IDs is entirely up to you, though the easiest way is to simply store those identifiers in the message header and then return them in the generator function:

```
return m.Header["dedupid"]
```

This plugin is under active development and will likely get more configurable options and features in the near future.