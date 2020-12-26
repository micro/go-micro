require 'net/http'
require 'json-rpc-objects/request'
require 'json-rpc-objects/response'

# RPC Client example in ruby
#
# This speaks directly to the service go.micro.srv.greeter

# set the host to the running go.micro.srv.greeter service
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
