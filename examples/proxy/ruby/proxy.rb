require 'net/http'
require 'json'

$registry_uri = URI("http://localhost:8081/registry")
$uri = URI("http://localhost:8081")

def register(service)
  http = Net::HTTP.new($registry_uri.host, $registry_uri.port)
  request = Net::HTTP::Post.new($registry_uri.request_uri)
  request.content_type = 'application/json'
  request.body = service.to_json
  http.request(request)
end

def deregister(service)
  http = Net::HTTP.new($registry_uri.host, $registry_uri.port)
  request = Net::HTTP::Delete.new($registry_uri.request_uri)
  request.content_type = 'application/json'
  request.body = service.to_json
  http.request(request)
end

def rpc_call(path, req)
  http = Net::HTTP.new($uri.host, $uri.port)
  request = Net::HTTP::Post.new(path)
  request.content_type = 'application/json'
  request.body = req.to_json
  JSON.parse(http.request(request).body)
end

def http_call(path, req)
  http = Net::HTTP.new($uri.host, $uri.port)
  request = Net::HTTP::Post.new(path)
  request.set_form_data(req)
  http.request(request).body
end
