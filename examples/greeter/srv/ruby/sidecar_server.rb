require 'webrick'
require 'securerandom'
require 'net/http'
require 'json'
require 'json-rpc-objects/request'
require 'json-rpc-objects/response'

# Sidecar JSON RPC Server
#
# An example service ruby.micro.srv.greeter 
# Registers with the sidecar 


register_uri = URI("http://localhost:8081/registry")
service = "ruby.micro.srv.greeter"
method = "Say.Hello"
host = "127.0.0.1"
port = 8080

server = WEBrick::HTTPServer.new :Port => port

server.mount_proc '/' do |req, res|
  request = JsonRpcObjects::Request::parse(req.body)
  response = request.class::version.response::create({:msg => "hello " + request.params[0]["name"]}, nil, :id => request.id)
  res.body = response.to_json
end

trap 'INT' do server.shutdown end

req = {
  "name" => service,
  "nodes" => [{
    "id" => service + "-" + SecureRandom.uuid,
    "address" => host,
    "port" => port
  }]
}

# register service
http = Net::HTTP.new(register_uri.host, register_uri.port)
request = Net::HTTP::Post.new(register_uri.request_uri)
request.content_type = 'application/json'
request.body = req.to_json
http.request(request)

server.start

# degister service
request = Net::HTTP::Delete.new(register_uri.request_uri)
request.content_type = 'application/json'
request.body = req.to_json
http.request(request)

