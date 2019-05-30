// Package docker builds docker images
package docker

import (
	"archive/tar"
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/fsouza/go-dockerclient"
	"github.com/micro/go-micro/runtime/package"
	"github.com/micro/go-micro/util/log"
)

type Packager struct {
	Options packager.Options
	Client  *docker.Client
}

func (d *Packager) Compile(s *packager.Source) (*packager.Binary, error) {
	image := filepath.Join(s.Repository.Path, s.Repository.Name)

	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	dockerFile := "Dockerfile"

	// open docker file
	f, err := os.Open(filepath.Join(s.Repository.Path, s.Repository.Name, dockerFile))
	if err != nil {
		return nil, err
	}
	// read docker file
	by, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	tarHeader := &tar.Header{
		Name: dockerFile,
		Size: int64(len(by)),
	}
	err = tw.WriteHeader(tarHeader)
	if err != nil {
		return nil, err
	}
	_, err = tw.Write(by)
	if err != nil {
		return nil, err
	}
	tr := bytes.NewReader(buf.Bytes())

	err = d.Client.BuildImage(docker.BuildImageOptions{
		Name:           image,
		Dockerfile:     dockerFile,
		InputStream:    tr,
		OutputStream:   ioutil.Discard,
		RmTmpContainer: true,
		SuppressOutput: true,
	})
	if err != nil {
		return nil, err
	}
	return &packager.Binary{
		Name:   image,
		Path:   image,
		Type:   "docker",
		Source: s,
	}, nil
}

func (d *Packager) Delete(b *packager.Binary) error {
	image := filepath.Join(b.Path, b.Name)
	return d.Client.RemoveImage(image)
}

func NewPackager(opts ...packager.Option) packager.Packager {
	options := packager.Options{}
	for _, o := range opts {
		o(&options)
	}
	endpoint := "unix:///var/run/docker.sock"
	client, err := docker.NewClient(endpoint)
	if err != nil {
		log.Fatal(err)
	}
	return &Packager{
		Options: options,
		Client:  client,
	}
}
