require './proxy'

puts rpc_call("/greeter/say/hello", {"name": "John"})
