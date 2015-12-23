package client

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"
	"time"

	log "github.com/golang/glog"

	ct "github.com/piemapping/contract/proto/trace"
	c "github.com/piemapping/go-micro/context"
	"github.com/piemapping/plugged/pubsub"
	"github.com/piemapping/plugged/trace"
	"github.com/piemapping/plugged/uuid"
)

var (
	tracePub     pubsub.PubSub
	traceChannel = make(chan trace.Trace, 200)
)

// Initialising our worker
func init() {
	go func() {
		for {
			select {
			case tr := <-traceChannel:
				if err := publishTrace(tr); err != nil {
					log.Warningf("Unable to publish trace [id=%s]: %v", tr.ID())
				}
			}
		}
	}()

	log.Info("Worker trace publisher initialised")
}

func createTrace(ctx context.Context, category trace.TraceType, to, endpoint string, payload interface{}, err error) (trace.Trace, context.Context) {
	md, _ := c.GetMetadata(ctx)
	from := md["from"]
	parentTraceId := md["X-parent-trace-Id"]

	uuider := uuid.NewPrefixedUUID(from)
	traceId := uuider.UUIDString()
	md["X-trace-id"] = traceId

	if len(parentTraceId) == 0 {
		parentTraceId = traceId
		md["X-parent-trace-Id"] = traceId
		ctx = c.WithMetadata(ctx, md)
	}

	tr := &trace.SimpleTrace{
		TraceID:       traceId,
		ParentTraceID: parentTraceId,
		// @todo get the address somehow
		Address:      "",
		FromService:  from,
		ToService:    to,
		EndpointName: endpoint,
		Category:     category,
		MillisTs:     time.Now().UnixNano() / 1000000,
		Err:          err,
	}

	pl, ok := payload.(proto.Message)
	if ok {
		tr.ProtoPayload = pl
	}

	return tr, ctx
}

func submitTrace(tr trace.Trace) error {
	// Asynchronously handle trace messages - without blocking!
	select {
	case traceChannel <- tr:
		return nil
	default:
		return fmt.Errorf("Unable to submit trace: %v", tr.ID())
	}
}

func publishTrace(tr trace.Trace) error {
	var err error
	if tracePub == nil {
		// lazily load our trace publisher
		tracePub, err = pubsub.NewPubSub("tracepub", 1)
		if err != nil {
			log.Errorf("Unable to create publisher for traces: %v", err)
			return err
		}
	}

	trType := ct.TraceType(tr.Type())

	// Now convert our trace into a proto
	tp := &ct.Trace{
		Id:              proto.String(tr.ID()),
		ParentId:        proto.String(tr.ParentID()),
		Host:            proto.String(tr.Host()),
		From:            proto.String(tr.From()),
		To:              proto.String(tr.To()),
		Endpoint:        proto.String(tr.Endpoint()),
		Type:            &trType,
		TimestampMillis: proto.Int64(tr.TimestampMillis()),
	}

	// Stringify the error
	if tr.Error() != nil {
		tp.Error = proto.String(fmt.Sprintf("%v", tr.Error))
	}

	// Check whether the underlying message is protobuf
	// If yes then get the compact string representation
	if protoMsg, ok := tr.Payload().(proto.Message); ok {
		tp.Payload = proto.String(protoMsg.String())
	}

	return tracePub.Publish("trace", tp)
}
