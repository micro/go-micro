// Package docker builds docker images
package docker

import (
	"archive/tar"
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/micro/go-micro/v3/build"
	"github.com/micro/go-micro/v3/logger"
)

type dockerBuild struct {
	Options build.Options
	Client  *docker.Client
}

func (d *dockerBuild) Package(name string, s *build.Source) (*build.Package, error) {
	image := name

	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	dockerFile := "Dockerfile"

	// open docker file
	f, err := os.Open(filepath.Join(s.Path, dockerFile))
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
	return &build.Package{
		Name:   image,
		Path:   image,
		Type:   "docker",
		Source: s,
	}, nil
}

func (d *dockerBuild) Remove(b *build.Package) error {
	return d.Client.RemoveImage(b.Name)
}

func (d *dockerBuild) String() string {
	return "docker"
}

func NewBuild(opts ...build.Option) build.Build {
	options := build.Options{}
	for _, o := range opts {
		o(&options)
	}
	endpoint := "unix:///var/run/docker.sock"
	client, err := docker.NewClient(endpoint)
	if err != nil {
		logger.Fatal(err)
	}
	return &dockerBuild{
		Options: options,
		Client:  client,
	}
}
