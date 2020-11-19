package protorpc

import (
	"encoding/binary"
	"io"
)

// WriteNetString writes data to a big-endian netstring on a Writer.
// Size is always a 32-bit unsigned int.
func WriteNetString(w io.Writer, data []byte) (written int, err error) {
	size := make([]byte, 4)
	binary.BigEndian.PutUint32(size, uint32(len(data)))
	if written, err = w.Write(size); err != nil {
		return
	}
	return w.Write(data)
}

// ReadNetString reads data from a big-endian netstring.
func ReadNetString(r io.Reader) (data []byte, err error) {
	sizeBuf := make([]byte, 4)
	_, err = r.Read(sizeBuf)
	if err != nil {
		return nil, err
	}
	size := binary.BigEndian.Uint32(sizeBuf)
	if size == 0 {
		return nil, nil
	}
	data = make([]byte, size)
	_, err = r.Read(data)
	if err != nil {
		return nil, err
	}
	return
}
