require 'webrick'
require 'json-rpc-objects/request'
require 'json-rpc-objects/response'

# JSON RPC Server
#
# An example service ruby.micro.srv.greeter

server = WEBrick::HTTPServer.new :Port => 8080

server.mount_proc '/' do |req, res|
  request = JsonRpcObjects::Request::parse(req.body)
  response = request.class::version.response::create({:msg => "hello " + request.params[0]["name"]})
  res.body = response.to_json
end

trap 'INT' do server.shutdown end

server.start

