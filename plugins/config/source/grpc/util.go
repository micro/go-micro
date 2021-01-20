package grpc

import (
	"time"

	"github.com/asim/go-micro/v3/config/source"
	proto "github.com/asim/go-micro/plugins/config/source/grpc/v3/proto"
)

func toChangeSet(c *proto.ChangeSet) *source.ChangeSet {
	return &source.ChangeSet{
		Data:      c.Data,
		Checksum:  c.Checksum,
		Format:    c.Format,
		Timestamp: time.Unix(c.Timestamp, 0),
		Source:    c.Source,
	}
}
