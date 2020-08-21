package local

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEntrypoint(t *testing.T) {
	wd, _ := os.Getwd()

	// test a service with idiomatic folder structure
	result, err := entrypoint(filepath.Join(wd, "test"))
	assert.Nil(t, err, "Didn'expected entrypoint to return an error")
	assert.Equal(t, "cmd/test/main.go", result, "Expected entrypoint to return cmd/test/main.go")

	// test a service with a top level main.go
	result, err = entrypoint(filepath.Join(wd, "test/bar"))
	assert.Nil(t, err, "Didn'expected entrypoint to return an error")
	assert.Equal(t, "main.go", result, "Expected entrypoint to return main.go")

	// test a service with multiple main.go files within the cmd folder
	result, err = entrypoint(filepath.Join(wd, "test/foo"))
	assert.Error(t, err, "Expected entrypoint to return an error when multiple main.go files exist")
	assert.Equal(t, "", result, "Expected entrypoint to not return a result")

	// test a service with no main.go files
	result, err = entrypoint(filepath.Join(wd, "test/empty"))
	assert.Error(t, err, "Expected entrypoint to return an error when no main.go files exist")
	assert.Equal(t, "", result, "Expected entrypoint to not return a result")
}
