package source

import (
	"crypto/md5"
	"fmt"
)

// Sum returns the md5 checksum of the ChangeSet data
func (c *ChangeSet) Sum() string {
	h := md5.New()
	h.Write(c.Data)
	return fmt.Sprintf("%x", h.Sum(nil))
}
