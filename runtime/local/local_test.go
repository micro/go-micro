package local

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEntrypoint(t *testing.T) {
	wd, _ := os.Getwd()

	result := entrypoint(filepath.Join(wd, "test"))
	assert.Equal(t, result, "cmd/main.go", "Expected entrypoint to return cmd/main.go")

	result = entrypoint(filepath.Join(wd, "test/foo"))
	assert.Equal(t, result, "main.go", "Expected entrypoint to return main.go")
}
