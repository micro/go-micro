require 'net/http'
require 'json'

# Sidecar Client example in ruby
#
# This speaks to the service go.micro.srv.greeter
# via the sidecar application HTTP interface

uri = URI("http://localhost:8081/rpc")

req = {
  "service" => "ruby.micro.srv.greeter",
  "method"  => "Say.Hello",
  "request" => {"name" => "John"}
}

# do request
http = Net::HTTP.new(uri.host, uri.port)
request = Net::HTTP::Post.new(uri.request_uri)
request.content_type = 'application/json'
request.body = req.to_json

puts JSON.parse(http.request(request).body)
