# Ruby Greeter Server

This is an example of how to run a ruby service within the Micro world.

## Simple RPC

examples/greeter/server/ruby/rpc_server.rb runs on localhost:8080 and responds to JSON RPC requests
examples/greeter/client/ruby/rpc_client.rb demonstrates a client request to this service

rpc_server.rb
```ruby
server = WEBrick::HTTPServer.new :Port => 8080

server.mount_proc '/' do |req, res|
  request = JsonRpcObjects::Request::parse(req.body)
  response = request.class::version.response::create({:msg => "hello " + request.params[0]["name"]})
  res.body = response.to_json
end

trap 'INT' do server.shutdown end

server.start
```

rpc_client.rb
```ruby
uri = URI("http://localhost:8080")
method = "Say.Hello"
request = {:name => "John"}

# create request
req = JsonRpcObjects::Request::create(method, [request], :id => 1)

# do request
http = Net::HTTP.new(uri.host, uri.port)
request = Net::HTTP::Post.new(uri.request_uri)
request.content_type = 'application/json'
request.body = req.to_json

# parse response
puts JsonRpcObjects::Response::parse(http.request(request).body).result["msg"]
```

## Sidecar Registration

By registering with discovery using the sidecar, other services can find and query your service.

An example server can be found at examples/greeter/server/ruby/sidecar_server.rb
An example client can be found at examples/greeter/client/ruby/sidecar_client.rb 

### Run the proxy
```shell
$ go get github.com/micro/micro
$ micro proxy
```

### Register
```ruby
# service definition
service = {
  "name" => service,
  "nodes" => [{
    "id" => service + "-" + SecureRandom.uuid,
    "address" => host,
    "port" => port
  }]
}

# registration url
register_uri = URI("http://localhost:8081/registry")

# register service
http = Net::HTTP.new(register_uri.host, register_uri.port)
request = Net::HTTP::Post.new(register_uri.request_uri)
request.content_type = 'application/json'
request.body = service.to_json
http.request(request)

# degister service
request = Net::HTTP::Delete.new(register_uri.request_uri)
request.content_type = 'application/json'
request.body = service.to_json
http.request(request)
```
