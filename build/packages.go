package build

import "fmt"

// Binary is an executable package
type Binary struct {
	// Path to the binary
	Path string
}

// Location of the binary
func (b *Binary) Location() string {
	return b.Path
}

// DockerImage is a container
type DockerImage struct {
	// Name of the image, e.g. micro/micro
	Name string
	// Repository containing the image, e.g. registry.hub.docker.com
	Repository string
}

// Location of the image
func (i *DockerImage) Location() string {
	if len(i.Repository) == 0 {
		return i.Name
	}

	return fmt.Sprintf("%v/%v", i.Repository, i.Name)
}
