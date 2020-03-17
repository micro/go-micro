// Package service implements the store service interface
package service

import (
	"context"
	"io"
	"time"

	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/errors"
	"github.com/micro/go-micro/v2/metadata"
	"github.com/micro/go-micro/v2/store"
	pb "github.com/micro/go-micro/v2/store/service/proto"
)

type serviceStore struct {
	options store.Options

	// Namespace to use
	Namespace string

	// Addresses of the nodes
	Nodes []string

	// Prefix to use
	Prefix string

	// Suffix to use
	Suffix string

	// store service client
	Client pb.StoreService
}

func (s *serviceStore) Init(opts ...store.Option) error {
	for _, o := range opts {
		o(&s.options)
	}
	s.Namespace = s.options.Namespace
	s.Prefix = s.options.Prefix
	s.Nodes = s.options.Nodes

	return nil
}

func (s *serviceStore) Context() context.Context {
	ctx := context.Background()

	md := make(metadata.Metadata)

	if len(s.Namespace) > 0 {
		md["Micro-Namespace"] = s.Namespace
	}

	if len(s.Prefix) > 0 {
		md["Micro-Prefix"] = s.Prefix
	}

	return metadata.NewContext(ctx, md)
}

// Sync all the known records
func (s *serviceStore) List(opts ...store.ListOption) ([]string, error) {
	stream, err := s.Client.List(s.Context(), &pb.ListRequest{}, client.WithAddress(s.Nodes...))
	if err != nil && errors.Equal(err, errors.NotFound("", "")) {
		return nil, store.ErrNotFound
	} else if err != nil {
		return nil, err
	}
	defer stream.Close()

	var keys []string

	for {
		rsp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return keys, err
		}

		for _, key := range rsp.Keys {
			keys = append(keys, key)
		}
	}

	return keys, nil
}

// Read a record with key
func (s *serviceStore) Read(key string, opts ...store.ReadOption) ([]*store.Record, error) {
	var options store.ReadOptions
	for _, o := range opts {
		o(&options)
	}

	rsp, err := s.Client.Read(s.Context(), &pb.ReadRequest{
		Key: key,
		Options: &pb.ReadOptions{
			Prefix: options.Prefix,
		},
	}, client.WithAddress(s.Nodes...))
	if err != nil && errors.Equal(err, errors.NotFound("", "")) {
		return nil, store.ErrNotFound
	} else if err != nil {
		return nil, err
	}

	records := make([]*store.Record, 0, len(rsp.Records))

	for _, val := range rsp.Records {
		records = append(records, &store.Record{
			Key:    val.Key,
			Value:  val.Value,
			Expiry: time.Duration(val.Expiry) * time.Second,
		})
	}

	return records, nil
}

// Write a record
func (s *serviceStore) Write(record *store.Record, opts ...store.WriteOption) error {
	_, err := s.Client.Write(s.Context(), &pb.WriteRequest{
		Record: &pb.Record{
			Key:    record.Key,
			Value:  record.Value,
			Expiry: int64(record.Expiry.Seconds()),
		},
	}, client.WithAddress(s.Nodes...))
	if err != nil && errors.Equal(err, errors.NotFound("", "")) {
		return store.ErrNotFound
	}

	return err
}

// Delete a record with key
func (s *serviceStore) Delete(key string, opts ...store.DeleteOption) error {
	_, err := s.Client.Delete(s.Context(), &pb.DeleteRequest{
		Key: key,
	}, client.WithAddress(s.Nodes...))
	if err != nil && errors.Equal(err, errors.NotFound("", "")) {
		return store.ErrNotFound
	}

	return err
}

func (s *serviceStore) String() string {
	return "service"
}

func (s *serviceStore) Options() store.Options {
	return s.options
}

// NewStore returns a new store service implementation
func NewStore(opts ...store.Option) store.Store {
	var options store.Options
	for _, o := range opts {
		o(&options)
	}

	service := &serviceStore{
		options:   options,
		Namespace: options.Namespace,
		Prefix:    options.Prefix,
		Nodes:     options.Nodes,
		Client:    pb.NewStoreService("go.micro.store", client.DefaultClient),
	}

	return service
}
