# Sharding

A sharding example using the greeter application and a X-From-User header as the sharding key

## Contents

- api.go - a modified version of the greeter api to include sharding

## Run 

### Greeter Service

Run multiple copies of the greeter

```
cd ../greeter
go run srv/main.go
```

### Greeter API

```
go run api.go
```

### Micro API

```
micro api
```

### Call API

Call the API with X-From-User header. Change the user to see the effects of sharding.

```shell
curl  -H "X-From-User: john" http://localhost:8080/greeter/say/hello?name=John
```

