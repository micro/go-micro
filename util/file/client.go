package file

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/micro/go-micro/v2/client"
	proto "github.com/micro/go-micro/v2/util/file/proto"
)

// Client is the client interface to access files
type File interface {
	Open(filename string, truncate bool) (int64, error)
	Stat(filename string) (*proto.StatResponse, error)
	GetBlock(sessionId, blockId int64) ([]byte, error)
	ReadAt(sessionId, offset, size int64) ([]byte, error)
	Read(sessionId int64, buf []byte) (int, error)
	Write(sessionId, offset int64, data []byte) error
	Close(sessionId int64) error
	Download(filename, saveFile string) error
	Upload(filename, localFile string) error
	DownloadAt(filename, saveFile string, blockId int) error
}

// NewClient returns a new Client which uses a micro Client
func New(service string, c client.Client) File {
	return &fc{proto.NewFileService(service, c)}
}

const (
	blockSize = 512 * 1024
)

type fc struct {
	c proto.FileService
}

func (c *fc) Open(filename string, truncate bool) (int64, error) {
	rsp, err := c.c.Open(context.TODO(), &proto.OpenRequest{
		Filename: filename,
		Truncate: truncate,
	})
	if err != nil {
		return 0, err
	}
	return rsp.Id, nil
}

func (c *fc) Stat(filename string) (*proto.StatResponse, error) {
	return c.c.Stat(context.TODO(), &proto.StatRequest{Filename: filename})
}

func (c *fc) GetBlock(sessionId, blockId int64) ([]byte, error) {
	return c.ReadAt(sessionId, blockId*blockSize, blockSize)
}

func (c *fc) ReadAt(sessionId, offset, size int64) ([]byte, error) {
	rsp, err := c.c.Read(context.TODO(), &proto.ReadRequest{Id: sessionId, Size: size, Offset: offset})
	if err != nil {
		return nil, err
	}

	if rsp.Eof {
		err = io.EOF
	}

	if rsp.Data == nil {
		rsp.Data = make([]byte, size)
	}

	if size != rsp.Size {
		return rsp.Data[:rsp.Size], err
	}

	return rsp.Data, nil
}

func (c *fc) Read(sessionId int64, buf []byte) (int, error) {
	b, err := c.ReadAt(sessionId, 0, int64(cap(buf)))
	if err != nil {
		return 0, err
	}
	copy(buf, b)
	return len(b), nil
}

func (c *fc) Write(sessionId, offset int64, data []byte) error {
	_, err := c.c.Write(context.TODO(), &proto.WriteRequest{
		Id:     sessionId,
		Offset: offset,
		Data:   data})
	return err
}

func (c *fc) Close(sessionId int64) error {
	_, err := c.c.Close(context.TODO(), &proto.CloseRequest{Id: sessionId})
	return err
}

func (c *fc) Download(filename, saveFile string) error {
	return c.DownloadAt(filename, saveFile, 0)
}

func (c *fc) Upload(filename, localFile string) error {
	file, err := os.Open(localFile)
	if err != nil {
		return err
	}
	defer file.Close()

	offset := 0
	sessionId, err := c.Open(filename, true)
	defer c.Close(sessionId)
	if err != nil {
		return err
	}
	reader := bufio.NewReader(file)
	part := make([]byte, blockSize)

	for {
		count, err := reader.Read(part)
		if err != nil {
			break
		}
		err = c.Write(sessionId, int64(offset), part)
		if err != nil {
			return err
		}
		offset += count
	}
	if err != nil && err != io.EOF {
		return fmt.Errorf("Error reading %v: %v", localFile, err)
	}
	return nil
}

func (c *fc) DownloadAt(filename, saveFile string, blockId int) error {
	stat, err := c.Stat(filename)
	if err != nil {
		return err
	}
	if stat.Type == "Directory" {
		return errors.New(fmt.Sprintf("%s is directory.", filename))
	}

	blocks := int(stat.Size / blockSize)
	if stat.Size%blockSize != 0 {
		blocks += 1
	}

	log.Printf("Download %s in %d blocks\n", filename, blocks-blockId)

	file, err := os.OpenFile(saveFile, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	sessionId, err := c.Open(filename, false)
	if err != nil {
		return err
	}

	for i := blockId; i < blocks; i++ {
		buf, rerr := c.GetBlock(sessionId, int64(i))
		if rerr != nil && rerr != io.EOF {
			return rerr
		}
		if _, werr := file.WriteAt(buf, int64(i)*blockSize); werr != nil {
			return werr
		}

		if i%((blocks-blockId)/100+1) == 0 {
			log.Printf("Downloading %s [%d/%d] blocks", filename, i-blockId+1, blocks-blockId)
		}

		if rerr == io.EOF {
			break
		}
	}
	log.Printf("Download %s completed", filename)

	c.Close(sessionId)

	return nil
}
