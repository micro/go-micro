// Package service implements the store service interface
package service

import (
	"context"
	"io"
	"time"

	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/config/options"
	"github.com/micro/go-micro/store"
	pb "github.com/micro/go-micro/store/service/proto"
)

type serviceStore struct {
	options.Options

	// Addresses of the nodes
	Nodes []string

	// store service client
	Client pb.StoreService
}

// Sync all the known records
func (s *serviceStore) List() ([]*store.Record, error) {
	stream, err := s.Client.List(context.Background(), &pb.ListRequest{}, client.WithAddress(s.Nodes...))
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	var records []*store.Record

	for {
		rsp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return records, err
		}
		for _, record := range rsp.Records {
			records = append(records, &store.Record{
				Key:    record.Key,
				Value:  record.Value,
				Expiry: time.Duration(record.Expiry) * time.Second,
			})
		}
	}

	return records, nil
}

// Read a record with key
func (s *serviceStore) Read(keys ...string) ([]*store.Record, error) {
	rsp, err := s.Client.Read(context.Background(), &pb.ReadRequest{
		Keys: keys,
	}, client.WithAddress(s.Nodes...))
	if err != nil {
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
func (s *serviceStore) Write(recs ...*store.Record) error {
	records := make([]*pb.Record, 0, len(recs))

	for _, record := range recs {
		records = append(records, &pb.Record{
			Key:    record.Key,
			Value:  record.Value,
			Expiry: int64(record.Expiry.Seconds()),
		})
	}

	_, err := s.Client.Write(context.Background(), &pb.WriteRequest{
		Records: records,
	}, client.WithAddress(s.Nodes...))

	return err
}

// Delete a record with key
func (s *serviceStore) Delete(keys ...string) error {
	_, err := s.Client.Delete(context.Background(), &pb.DeleteRequest{
		Keys: keys,
	}, client.WithAddress(s.Nodes...))
	return err
}

// NewStore returns a new store service implementation
func NewStore(opts ...options.Option) store.Store {
	options := options.NewOptions(opts...)

	var nodes []string

	n, ok := options.Values().Get("store.nodes")
	if ok {
		nodes = n.([]string)
	}

	service := &serviceStore{
		Options: options,
		Nodes:   nodes,
		Client:  pb.NewStoreService("go.micro.store", client.DefaultClient),
	}

	return service
}
