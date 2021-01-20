package memcached

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	mc "github.com/bradfitz/gomemcache/memcache"
	log "github.com/asim/go-micro/v3/logger"
	"github.com/asim/go-micro/v3/store"
)

type mkv struct {
	options store.Options
	Server  *mc.ServerList
	Client  *mc.Client
}

func (m *mkv) Init(opts ...store.Option) error {
	for _, o := range opts {
		o(&m.options)
	}
	return m.configure()
}

func (m *mkv) Options() store.Options {
	return m.options
}

func (m *mkv) Close() error {
	// memcaced does not supports close?
	return nil
}

func (m *mkv) Read(key string, opts ...store.ReadOption) ([]*store.Record, error) {
	// TODO: implement read options
	records := make([]*store.Record, 0, 1)

	keyval, err := m.Client.Get(key)
	if err != nil && err == mc.ErrCacheMiss {
		return nil, store.ErrNotFound
	} else if err != nil {
		return nil, err
	}

	if keyval == nil {
		return nil, store.ErrNotFound
	}

	records = append(records, &store.Record{
		Key:    keyval.Key,
		Value:  keyval.Value,
		Expiry: time.Second * time.Duration(keyval.Expiration),
	})

	return records, nil
}

func (m *mkv) Delete(key string, opts ...store.DeleteOption) error {
	return m.Client.Delete(key)
}

func (m *mkv) Write(record *store.Record, opts ...store.WriteOption) error {
	return m.Client.Set(&mc.Item{
		Key:        record.Key,
		Value:      record.Value,
		Expiration: int32(record.Expiry.Seconds()),
	})
}

func (m *mkv) List(opts ...store.ListOption) ([]string, error) {
	// stats
	// cachedump
	// get keys

	var keys []string

	//store := make(map[string]string)
	if err := m.Server.Each(func(c net.Addr) error {
		cc, err := net.Dial("tcp", c.String())
		if err != nil {
			return err
		}
		defer cc.Close()

		b := bufio.NewReadWriter(bufio.NewReader(cc), bufio.NewWriter(cc))

		// get records
		if _, err := fmt.Fprintf(b, "stats records\r\n"); err != nil {
			return err
		}

		b.Flush()

		v, err := b.ReadSlice('\n')
		if err != nil {
			return err
		}

		parts := bytes.Split(v, []byte("\n"))
		if len(parts) < 1 {
			return nil
		}
		vals := strings.Split(string(parts[0]), ":")
		records := vals[1]

		// drain
		for {
			buf, err := b.ReadSlice('\n')
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			if strings.HasPrefix(string(buf), "END") {
				break
			}
		}

		b.Writer.Reset(cc)
		b.Reader.Reset(cc)

		if _, err := fmt.Fprintf(b, "lru_crawler metadump %s\r\n", records); err != nil {
			return err
		}
		b.Flush()

		for {
			v, err := b.ReadString('\n')
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			if strings.HasPrefix(v, "END") {
				break
			}
			key := strings.Split(v, " ")[0]
			keys = append(keys, strings.TrimPrefix(key, "key="))
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return keys, nil
}

func (m *mkv) String() string {
	return "memcached"
}

func NewStore(opts ...store.Option) store.Store {
	var options store.Options
	for _, o := range opts {
		o(&options)
	}

	s := new(mkv)
	s.options = options

	if err := s.configure(); err != nil {
		log.Fatal(err)
	}

	return s
}

func (m *mkv) configure() error {
	nodes := m.options.Nodes

	if len(nodes) == 0 {
		nodes = []string{"127.0.0.1:11211"}
	}

	ss := new(mc.ServerList)
	ss.SetServers(nodes...)

	m.Server = ss
	m.Client = mc.New(nodes...)

	return nil
}
