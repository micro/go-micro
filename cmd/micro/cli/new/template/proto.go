package template

var (
	ProtoSRV = `syntax = "proto3";

package {{dehyphen .Alias}};

option go_package = "./proto;{{dehyphen .Alias}}";

service {{title .Alias}} {
	rpc Call(Request) returns (Response) {}
	rpc Stream(StreamingRequest) returns (stream StreamingResponse) {}
}

message Message {
	string say = 1;
}

message Request {
	// Name of the person to greet
	string name = 1;
}

message Response {
	// Greeting message
	string msg = 1;
}

message StreamingRequest {
	// Number of responses to stream back
	int64 count = 1;
}

message StreamingResponse {
	// Current sequence number in the stream
	int64 count = 1;
}
`
)
