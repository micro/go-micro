package template

// ProtoFNC is the .proto file template used for new function projects.
var ProtoFNC = `syntax = "proto3";

package {{dehyphen .Service}};

option go_package = "./proto;{{dehyphen .Service}}";

service {{title .Service}} {
	rpc Call(CallRequest) returns (CallResponse) {}
}

message CallRequest {
	string name = 1;
}

message CallResponse {
	string msg = 1;
}
`

// ProtoSRV is the .proto file template used for new service projects.
var ProtoSRV = `syntax = "proto3";

package {{dehyphen .Service}};

option go_package = "./proto;{{dehyphen .Service}}";

service {{title .Service}} {
	rpc Call(CallRequest) returns (CallResponse) {}
	rpc ClientStream(stream ClientStreamRequest) returns (ClientStreamResponse) {}
	rpc ServerStream(ServerStreamRequest) returns (stream ServerStreamResponse) {}
	rpc BidiStream(stream BidiStreamRequest) returns (stream BidiStreamResponse) {}
}

message CallRequest {
	string name = 1;
}

message CallResponse {
	string msg = 1;
}

message ClientStreamRequest {
	int64 stroke = 1;
}

message ClientStreamResponse {
	int64 count = 1;
}

message ServerStreamRequest {
	int64 count = 1;
}

message ServerStreamResponse {
	int64 count = 1;
}

message BidiStreamRequest {
	int64 stroke = 1;
}

message BidiStreamResponse {
	int64 stroke = 1;
}
`
