# Greeter 问候示例服务

本示例展示基于gRPC的Go-Micro服务

## 本目录有

- **server** - Greeter的gRPC服务端
- **client** - gRPC客户端，会调用一次server
- **function** - 演示gRPC Greeter function接口，更多关于Function，请查阅[Function](https://micro.mu/docs/writing-a-go-function_cn.html)
- **gateway** - gRPC网关

## 测试服务

运行服务

```
$ go run server/main.go --registry=mdns
2016/11/03 18:41:22 Listening on [::]:55194
2016/11/03 18:41:22 Broker Listening on [::]:55195
2016/11/03 18:41:22 Registering node: go.micro.srv.greeter-1e200612-a1f5-11e6-8e84-68a86d0d36b6
```

测试

```
$ go run client/main.go --registry=mdns
Hello John
```

## 测试 Function

运行测试

```
go run function/main.go --registry=mdns
```

调用服务

服务端的Function服务只会执行一次，所以在下面的命令执行且服务端返回请求后，服务端便后退出

```bash
$ go run client/main.go --registry=mdns --service_name="go.micro.fnc.greeter"

# 返回
Hello John

# 再次执行
$ go run client/main.go --registry=mdns --service_name="go.micro.fnc.greeter"

# 就会报异常，找不到服务
{"id":"go.micro.client","code":500,"detail":"none available","status":"Internal Server Error"}

```

## 测试网关

指定地址再运行服务端：

```
go run server/main.go --registry=mdns --server_address=localhost:9090
```

运行网关

```
go run gateway/main.go
```

使用curl调用网关

```
curl -d '{"name": "john"}' http://localhost:8080/greeter/hello
# 返回
{"msg":"Hello john"}
```