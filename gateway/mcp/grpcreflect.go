package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	reflectionpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

// ReflectedGRPCTarget describes an external gRPC server whose reflection
// catalog should be exposed as MCP tools. It is intentionally opt-in: teams can
// bridge existing reflected gRPC services without changing their servers or
// registering them in go-micro.
type ReflectedGRPCTarget struct {
	// Name prefixes generated tools. When empty, Address is sanitized and used.
	Name string
	// Address is the host:port of the reflected gRPC server.
	Address string
	// DialOptions customize the connection. If none are supplied, an insecure
	// transport is used for local/dev interoperability.
	DialOptions []grpc.DialOption
	// Timeout bounds reflection discovery and individual tool calls.
	Timeout time.Duration
}

func (s *Server) discoverReflectedGRPC() error {
	for _, target := range s.opts.ReflectedGRPCTargets {
		if strings.TrimSpace(target.Address) == "" {
			continue
		}
		tools, err := s.reflectedGRPCTools(target)
		if err != nil {
			return err
		}
		for _, tool := range tools {
			s.tools[tool.Name] = tool
		}
	}
	return nil
}

func (s *Server) reflectedGRPCTools(target ReflectedGRPCTarget) ([]*Tool, error) {
	timeout := target.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(s.opts.Context, timeout)
	defer cancel()

	dialOpts := target.DialOptions
	if len(dialOpts) == 0 {
		dialOpts = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	}
	conn, err := grpc.NewClient(target.Address, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("connect reflected grpc target %s: %w", target.Address, err)
	}
	defer conn.Close()

	files, services, err := loadReflectedFiles(ctx, conn)
	if err != nil {
		return nil, fmt.Errorf("reflect grpc target %s: %w", target.Address, err)
	}

	prefix := target.Name
	if prefix == "" {
		prefix = sanitizeToolPart(target.Address)
	}

	var out []*Tool
	for _, serviceName := range services {
		desc, err := files.FindDescriptorByName(protoreflect.FullName(serviceName))
		if err != nil {
			continue
		}
		svc, ok := desc.(protoreflect.ServiceDescriptor)
		if !ok {
			continue
		}
		for i := 0; i < svc.Methods().Len(); i++ {
			method := svc.Methods().Get(i)
			if method.IsStreamingClient() || method.IsStreamingServer() {
				continue
			}
			fullMethod := "/" + string(svc.FullName()) + "/" + string(method.Name())
			toolName := prefix + "." + strings.ReplaceAll(string(svc.FullName()), ".", "_") + "." + string(method.Name())
			input := method.Input()
			out = append(out, &Tool{
				Name:        toolName,
				Description: fmt.Sprintf("Call reflected gRPC method %s on %s", fullMethod, target.Address),
				InputSchema: protoMessageSchema(input),
				Handler:     reflectedGRPCHandler(target, fullMethod, input, method.Output()),
			})
		}
	}
	return out, nil
}

func loadReflectedFiles(ctx context.Context, conn *grpc.ClientConn) (*protoregistryFiles, []string, error) {
	client := reflectionpb.NewServerReflectionClient(conn)
	stream, err := client.ServerReflectionInfo(ctx)
	if err != nil {
		return nil, nil, err
	}
	if err := stream.Send(&reflectionpb.ServerReflectionRequest{MessageRequest: &reflectionpb.ServerReflectionRequest_ListServices{ListServices: ""}}); err != nil {
		return nil, nil, err
	}
	resp, err := stream.Recv()
	if err != nil {
		return nil, nil, err
	}
	list := resp.GetListServicesResponse()
	if list == nil {
		return nil, nil, fmt.Errorf("reflection list services returned %T", resp.MessageResponse)
	}

	set := &descriptorpb.FileDescriptorSet{}
	seen := map[string]bool{}
	var services []string
	for _, svc := range list.Service {
		name := svc.Name
		if strings.HasPrefix(name, "grpc.reflection.") {
			continue
		}
		services = append(services, name)
		if err := requestFileContainingSymbol(ctx, client, name, set, seen); err != nil {
			return nil, nil, err
		}
	}
	files, err := newProtoregistryFiles(set)
	if err != nil {
		return nil, nil, err
	}
	return files, services, nil
}

func requestFileContainingSymbol(ctx context.Context, client reflectionpb.ServerReflectionClient, symbol string, set *descriptorpb.FileDescriptorSet, seen map[string]bool) error {
	stream, err := client.ServerReflectionInfo(ctx)
	if err != nil {
		return err
	}
	if err := stream.Send(&reflectionpb.ServerReflectionRequest{MessageRequest: &reflectionpb.ServerReflectionRequest_FileContainingSymbol{FileContainingSymbol: symbol}}); err != nil {
		return err
	}
	resp, err := stream.Recv()
	if err != nil {
		return err
	}
	fd := resp.GetFileDescriptorResponse()
	if fd == nil {
		return fmt.Errorf("reflection lookup for %s returned %T", symbol, resp.MessageResponse)
	}
	for _, raw := range fd.FileDescriptorProto {
		var file descriptorpb.FileDescriptorProto
		if err := proto.Unmarshal(raw, &file); err != nil {
			return err
		}
		name := file.GetName()
		if !seen[name] {
			seen[name] = true
			set.File = append(set.File, &file)
		}
	}
	return nil
}

// protoregistryFiles is a narrow wrapper that keeps imports local to this file.
type protoregistryFiles struct{ files *protoregistry.Files }

func newProtoregistryFiles(set *descriptorpb.FileDescriptorSet) (*protoregistryFiles, error) {
	files, err := protodesc.NewFiles(set)
	if err != nil {
		return nil, err
	}
	return &protoregistryFiles{files: files}, nil
}

func (p *protoregistryFiles) FindDescriptorByName(name protoreflect.FullName) (protoreflect.Descriptor, error) {
	return p.files.FindDescriptorByName(name)
}

func reflectedGRPCHandler(target ReflectedGRPCTarget, fullMethod string, input, output protoreflect.MessageDescriptor) func(map[string]interface{}) (interface{}, error) {
	return func(args map[string]interface{}) (interface{}, error) {
		timeout := target.Timeout
		if timeout == 0 {
			timeout = 10 * time.Second
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		dialOpts := target.DialOptions
		if len(dialOpts) == 0 {
			dialOpts = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
		}
		conn, err := grpc.NewClient(target.Address, dialOpts...)
		if err != nil {
			return nil, err
		}
		defer conn.Close()

		req := dynamicpb.NewMessage(input)
		raw, err := json.Marshal(args)
		if err != nil {
			return nil, err
		}
		if err := protojson.Unmarshal(raw, req); err != nil {
			return nil, err
		}
		rsp := dynamicpb.NewMessage(output)
		if err := conn.Invoke(ctx, fullMethod, req, rsp); err != nil {
			return nil, err
		}
		b, err := protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: true}.Marshal(rsp)
		if err != nil {
			return nil, err
		}
		var out interface{}
		if err := json.Unmarshal(b, &out); err != nil {
			return nil, err
		}
		return out, nil
	}
}

func protoMessageSchema(msg protoreflect.MessageDescriptor) map[string]interface{} {
	schema := map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
	props := schema["properties"].(map[string]interface{})
	fields := msg.Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		props[string(field.JSONName())] = protoFieldSchema(field)
	}
	return schema
}

func protoFieldSchema(field protoreflect.FieldDescriptor) map[string]interface{} {
	schema := map[string]interface{}{"type": protoJSONType(field)}
	if field.IsList() {
		schema["items"] = map[string]interface{}{"type": protoJSONType(field)}
	}
	if field.Kind() == protoreflect.MessageKind || field.Kind() == protoreflect.GroupKind {
		schema = protoMessageSchema(field.Message())
	}
	return schema
}

func protoJSONType(field protoreflect.FieldDescriptor) string {
	if field.IsList() {
		return "array"
	}
	switch field.Kind() {
	case protoreflect.BoolKind:
		return "boolean"
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Uint32Kind, protoreflect.Fixed32Kind, protoreflect.Int64Kind,
		protoreflect.Sint64Kind, protoreflect.Sfixed64Kind, protoreflect.Uint64Kind,
		protoreflect.Fixed64Kind:
		return "integer"
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		return "number"
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return "object"
	default:
		return "string"
	}
}

func sanitizeToolPart(s string) string {
	r := strings.NewReplacer(":", "_", "/", "_", ".", "_", "-", "_")
	return r.Replace(s)
}
