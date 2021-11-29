package mucp

import (
	"time"

	"go-micro.dev/v4/config/source"
	proto "github.com/asim/go-micro/plugins/config/source/mucp/v4/proto"
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
