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

	// The database to use
	Database string

	// The table to use
	Table string

	// Addresses of the nodes
	Nodes []string

	// store service client
	Client pb.StoreService
}

func (s *serviceStore) Close() error {
	return nil
}

func (s *serviceStore) Init(opts ...store.Option) error {
	for _, o := range opts {
		o(&s.options)
	}
	s.Database = s.options.Database
	s.Table = s.options.Table
	s.Nodes = s.options.Nodes

	return nil
}

func (s *serviceStore) Context() context.Context {
	ctx := context.Background()
	md := make(metadata.Metadata)
	if len(s.Database) > 0 {
		md["Micro-Database"] = s.Database
	}

	if len(s.Table) > 0 {
		md["Micro-Table"] = s.Table
	}
	return metadata.NewContext(ctx, md)
}

// Sync all the known records
func (s *serviceStore) List(opts ...store.ListOption) ([]string, error) {
	options := store.ListOptions{
		Database: s.Database,
		Table:    s.Table,
	}

	for _, o := range opts {
		o(&options)
	}

	listOpts := &pb.ListOptions{
		Database: options.Database,
		Table:    options.Table,
		Prefix:   options.Prefix,
		Suffix:   options.Suffix,
		Limit:    uint64(options.Limit),
		Offset:   uint64(options.Offset),
	}

	stream, err := s.Client.List(s.Context(), &pb.ListRequest{Options: listOpts}, client.WithAddress(s.Nodes...))
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
	options := store.ReadOptions{
		Database: s.Database,
		Table:    s.Table,
	}

	for _, o := range opts {
		o(&options)
	}

	readOpts := &pb.ReadOptions{
		Database: options.Database,
		Table:    options.Table,
		Prefix:   options.Prefix,
		Suffix:   options.Suffix,
		Limit:    uint64(options.Limit),
		Offset:   uint64(options.Offset),
	}

	rsp, err := s.Client.Read(s.Context(), &pb.ReadRequest{
		Key:     key,
		Options: readOpts,
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
	options := store.WriteOptions{
		Database: s.Database,
		Table:    s.Table,
	}

	for _, o := range opts {
		o(&options)
	}

	writeOpts := &pb.WriteOptions{
		Database: options.Database,
		Table:    options.Table,
	}

	_, err := s.Client.Write(s.Context(), &pb.WriteRequest{
		Record: &pb.Record{
			Key:    record.Key,
			Value:  record.Value,
			Expiry: int64(record.Expiry.Seconds()),
		},
		Options: writeOpts}, client.WithAddress(s.Nodes...))
	if err != nil && errors.Equal(err, errors.NotFound("", "")) {
		return store.ErrNotFound
	}

	return err
}

// Delete a record with key
func (s *serviceStore) Delete(key string, opts ...store.DeleteOption) error {
	options := store.DeleteOptions{
		Database: s.Database,
		Table:    s.Table,
	}

	for _, o := range opts {
		o(&options)
	}

	deleteOpts := &pb.DeleteOptions{
		Database: options.Database,
		Table:    options.Table,
	}

	_, err := s.Client.Delete(s.Context(), &pb.DeleteRequest{
		Key:     key,
		Options: deleteOpts,
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
		options:  options,
		Database: options.Database,
		Table:    options.Table,
		Nodes:    options.Nodes,
		Client:   pb.NewStoreService("go.micro.store", client.DefaultClient),
	}

	return service
}
