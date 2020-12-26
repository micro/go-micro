# Event

This is an example of using the micro API as an event gateway with the event handler

A http request is formatted as an [event](https://github.com/micro/go-api/blob/master/proto/api.proto#L28L39) and published on the go-micro message broker.

## Contents

- srv - A service which subscribes to events

## Usage

Run the micro api with the event handler set and with a namespace which used as part of the topic name

```
micro api --handler=event --namespace=go.micro.evt
```

Run the service

```
go run srv/main.go
```

### Event format 

On the receiving end the message will be formatted like so:

```
// A HTTP event as RPC
message Event {
	// e.g login
	string name = 1;
	// uuid
	string id = 2;
	// unix timestamp of event
	int64 timestamp = 3;
	// event headers
        map<string, Pair> header = 4;
	// the event data
	string data = 5;
}
```

### Publish Event

Publishing an event is as simple as making a http post request

```
curl -d '{"name": "john"}' http://localhost:8080/user/login
```

This request will be published to the topic `go.micro.evt.user` with event name `login`

### Receiving Event

A subscriber should be registered with the service for topic `go.micro.evt.user`

The subscriber should take the proto.Event type. See srv/main.go for the code.

The event received will look like the following

```
{
	name: "user.login",
	id: "go.micro.evt.user-user.login-693116e7-f20c-11e7-96c7-f40f242f6897",
	timestamp:1515152077,
	header: {...},
	data: {"name": "john"} 
}
```
