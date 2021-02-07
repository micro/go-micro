package zookeeper

import (
	"encoding/json"
	"path"
	"strings"

	"github.com/asim/go-micro/v3/registry"
	"github.com/go-zookeeper/zk"
)

func encode(s *registry.Service) ([]byte, error) {
	return json.Marshal(s)
}

func decode(ds []byte) (*registry.Service, error) {
	var s *registry.Service
	err := json.Unmarshal(ds, &s)
	return s, err
}

func nodePath(s, id string) string {
	service := strings.Replace(s, "/", "-", -1)
	node := strings.Replace(id, "/", "-", -1)
	return path.Join(prefix, service, node)
}

func childPath(parent, child string) string {
	return path.Join(parent, strings.Replace(child, "/", "-", -1))
}

func servicePath(s string) string {
	return path.Join(prefix, strings.Replace(s, "/", "-", -1))
}

func createPath(path string, data []byte, client *zk.Conn) error {
	exists, _, err := client.Exists(path)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	name := "/"
	p := strings.Split(path, "/")

	for _, v := range p[1 : len(p)-1] {
		name += v
		e, _, _ := client.Exists(name)
		if !e {
			_, err = client.Create(name, []byte{}, int32(0), zk.WorldACL(zk.PermAll))
			if err != nil {
				return err
			}
		}
		name += "/"
	}

	_, err = client.Create(path, data, int32(0), zk.WorldACL(zk.PermAll))
	return err
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
