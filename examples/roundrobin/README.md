# Round Robin

An example of using a round robin client wrapper with the greeter application. 

## Contents

- api.go - a modified version of the greeter api to include roundrobin

### Micro

```
go get github.com/micro/micro
```

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

```shell
curl  http://localhost:8080/greeter/say/hello?name=John
```

