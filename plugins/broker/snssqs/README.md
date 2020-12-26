# SQS SNS Broker Plugin for go-micro
Amazon Simple Notification Service and Simple Queue Service broker plugin for `go-micro` allows you to publish to SNS and subscribe messages brokered by SQS. This plugin _does not_ (yet) support automatic creation of SQS queues or SNS topics so those will have to exist in your infrastructure before attempting to send/receive.

## AWS Credentials
This plugin uses the official Go SDK for AWS. As such, it will obtain AWS credentials the same way all other `aws-go-sdk` applications do. The plugin explicitly allows the use of the shared credentials file to make development on workstations easier, but you can also supply the usual `AWS_*` environment variables in dev/test/prod environments. Also if you're deploying in EC2/ECS, the `IAM Role` will be picked up automatically and you won't need to supply any credentials.

## Publishing and Subscribing
Publishing and subscribing with the plugin should work just like all other brokers. Simply supply the name of the queue in the publish or subscribe arguments:

```go
broker.Publish("my_topic", msg)
...
broker.Subscribe("queue", subscriberFunc)
```

You will need to supply the `AWS_REGION` environment variable to configure the region (in addition to credentials as above).

Because SNS can't deliver to `FIFO` queues, you cannot subscribe to a `FIFO` queue using this broker.

## Options
If you're using a regular (non-fifo) queue you should be able to get by without having to supply any special options. However, if you need to specify a group identifier for a message or a de-duplication identifier, then you'll have to specify a generator function for those.

This plugin is under active development and will likely get more configurable options and features in the near future.