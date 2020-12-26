require './proxy'
require 'securerandom'
require 'webrick'

$service = {
  "name" => "go.micro.srv.greeter",
  "nodes" => [{
    "id" => "go.micro.srv.greeter-" + SecureRandom.uuid,
    "address" => "localhost",
    "port" => 4000
  }]
}

trap 'INT' do
  deregister($service)
  exit
end

# create server
server = WEBrick::HTTPServer.new :Port => 4000

# serve method Say.Hello
server.mount_proc '/greeter' do |req, res|
  res.body = "Hello #{req.query['name']}!"
end

# register service
register($service)

# start the server and block
server.start
