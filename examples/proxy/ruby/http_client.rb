require './proxy'

puts http_call("/greeter", {"name" => "John"})
