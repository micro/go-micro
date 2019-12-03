package handler

import (
	"context"
	"io"
	"time"

	"github.com/micro/go-micro/errors"
	"github.com/micro/go-micro/store"
	pb "github.com/micro/go-micro/store/service/proto"
)

type Store struct {
	Store store.Store
}

func (s *Store) Read(ctx context.Context, req *pb.ReadRequest, rsp *pb.ReadResponse) error {
	vals, err := s.Store.Read(req.Keys...)
	if err != nil {
		return errors.InternalServerError("go.micro.store", err.Error())
	}
	for _, val := range vals {
		rsp.Records = append(rsp.Records, &pb.Record{
			Key:    val.Key,
			Value:  val.Value,
			Expiry: int64(val.Expiry.Seconds()),
		})
	}
	return nil
}

func (s *Store) Write(ctx context.Context, req *pb.WriteRequest, rsp *pb.WriteResponse) error {
	records := make([]*store.Record, 0, len(req.Records))

	for _, record := range req.Records {
		records = append(records, &store.Record{
			Key:    record.Key,
			Value:  record.Value,
			Expiry: time.Duration(record.Expiry) * time.Second,
		})
	}

	err := s.Store.Write(records...)
	if err != nil {
		return errors.InternalServerError("go.micro.store", err.Error())
	}
	return nil
}

func (s *Store) Delete(ctx context.Context, req *pb.DeleteRequest, rsp *pb.DeleteResponse) error {
	err := s.Store.Delete(req.Keys...)
	if err != nil {
		return errors.InternalServerError("go.micro.store", err.Error())
	}
	return nil
}

func (s *Store) List(ctx context.Context, req *pb.ListRequest, stream pb.Store_ListStream) error {
	var vals []*store.Record
	var err error

	if len(req.Key) > 0 {
		vals, err = s.Store.Read(req.Key)
	} else {
		vals, err = s.Store.List()
	}
	if err != nil {
		return errors.InternalServerError("go.micro.store", err.Error())
	}
	rsp := new(pb.ListResponse)

	// TODO: batch sync
	for _, val := range vals {
		rsp.Records = append(rsp.Records, &pb.Record{
			Key:    val.Key,
			Value:  val.Value,
			Expiry: int64(val.Expiry.Seconds()),
		})
	}

	err = stream.Send(rsp)
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return errors.InternalServerError("go.micro.store", err.Error())
	}
	return nil
}
