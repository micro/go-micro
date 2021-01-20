package awsxray

import (
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"context"

	"github.com/asim/go-awsxray"
	"github.com/micro/go-micro/v2/errors"
	"github.com/micro/go-micro/v2/metadata"
)

// getHTTP returns a http struct
func getHTTP(url, method string, err error) *awsxray.HTTP {
	return &awsxray.HTTP{
		Request: &awsxray.Request{
			Method: method,
			URL:    url,
		},
		Response: &awsxray.Response{
			Status: getStatus(err),
		},
	}
}

// getRandom generates a random byte slice
func getRandom(i int) []byte {
	b := make([]byte, i)
	for {
		// keep trying till we get it
		if _, err := rand.Read(b); err != nil {
			continue
		}
		return b
	}
}

// getSegment creates a new segment based on whether we're part of an existing flow
func getSegment(name string, ctx context.Context) *awsxray.Segment {
	md, _ := metadata.FromContext(ctx)
	parentId := getParentId(md)
	traceId := getTraceId(md)

	// try get existing segment for parent Id
	if s, ok := awsxray.FromContext(ctx); ok {
		// only set existing segment as parent if its not a subsegment itself
		if len(parentId) == 0 && len(s.Type) == 0 {
			parentId = s.Id
		}
		if len(traceId) == 0 {
			traceId = s.TraceId
		}
	}

	// create segment
	s := &awsxray.Segment{
		Id:        fmt.Sprintf("%x", getRandom(8)),
		Name:      name,
		TraceId:   traceId,
		StartTime: float64(time.Now().Truncate(time.Millisecond).UnixNano()) / 1e9,
	}

	// we have a parent so subsegment
	if len(parentId) > 0 {
		s.ParentId = parentId
		s.Type = "subsegment"
	}

	return s
}

// getStatus returns a status code from the error
func getStatus(err error) int {
	// no error
	if err == nil {
		return 200
	}

	// try get errors.Error
	if e, ok := err.(*errors.Error); ok {
		return int(e.Code)
	}

	// try parse marshalled error
	if e := errors.Parse(err.Error()); e.Code > 0 {
		return int(e.Code)
	}

	// could not parse, 500
	return 500
}

// getTraceId returns trace header or generates a new one
func getTraceId(md metadata.Metadata) string {
	// try as is
	if h, ok := md[awsxray.TraceHeader]; ok {
		return awsxray.GetTraceId(h)
	}

	// try lower case
	if h, ok := md[strings.ToLower(awsxray.TraceHeader)]; ok {
		return awsxray.GetTraceId(h)
	}

	// generate new one, probably a bad idea...
	return fmt.Sprintf("%d-%x-%x", 1, time.Now().Unix(), getRandom(12))
}

// getParentId returns parent header or blank
func getParentId(md metadata.Metadata) string {
	// try as is
	if h, ok := md[awsxray.TraceHeader]; ok {
		return awsxray.GetParentId(h)
	}

	// try lower case
	if h, ok := md[strings.ToLower(awsxray.TraceHeader)]; ok {
		return awsxray.GetParentId(h)
	}

	return ""
}

func newXRay(opts Options) *awsxray.AWSXRay {
	return awsxray.New(
		awsxray.WithClient(opts.Client),
		awsxray.WithDaemon(opts.Daemon),
	)
}

func record(x *awsxray.AWSXRay, s *awsxray.Segment) error {
	// set end time
	s.EndTime = float64(time.Now().Truncate(time.Millisecond).UnixNano()) / 1e9
	return x.Record(s)
}

// setCallStatus sets the http section and related status
func setCallStatus(s *awsxray.Segment, url, method string, err error) {
	s.HTTP = getHTTP(url, method, err)

	status := getStatus(err)
	switch {
	case status >= 500:
		s.Fault = true
	case status >= 400:
		s.Error = true
	case err != nil:
		s.Fault = true
	}
}

func newContext(ctx context.Context, s *awsxray.Segment) context.Context {
	md, ok := metadata.FromContext(ctx)
	if !ok {
		md = make(map[string]string)
	}

	// set trace id in header
	md[awsxray.TraceHeader] = awsxray.SetTraceId(md[awsxray.TraceHeader], s.TraceId)
	// set parent id in header
	md[awsxray.TraceHeader] = awsxray.SetParentId(md[awsxray.TraceHeader], s.ParentId)
	// store segment in context
	ctx = awsxray.NewContext(ctx, s)
	// store metadata in context
	ctx = metadata.NewContext(ctx, md)

	return ctx
}
