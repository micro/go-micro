package file

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/micro/go-micro/v3/store"
	"github.com/stretchr/testify/assert"
)

func TestBlobStore(t *testing.T) {
	blob, err := NewBlobStore()
	assert.NotNilf(t, blob, "Blob should not be nil")
	assert.Nilf(t, err, "Error should be nil")

	t.Run("ReadMissingKey", func(t *testing.T) {
		res, err := blob.Read("")
		assert.Equal(t, store.ErrMissingKey, err, "Error should be missing key")
		assert.Nil(t, res, "Result should be nil")
	})

	t.Run("ReadNotFound", func(t *testing.T) {
		res, err := blob.Read("foo")
		assert.Equal(t, store.ErrNotFound, err, "Error should be not found")
		assert.Nil(t, res, "Result should be nil")
	})

	t.Run("WriteMissingKey", func(t *testing.T) {
		buf := bytes.NewBuffer([]byte("HelloWorld"))
		err := blob.Write("", buf)
		assert.Equal(t, store.ErrMissingKey, err, "Error should be missing key")
	})

	t.Run("WriteValid", func(t *testing.T) {
		buf := bytes.NewBuffer([]byte("world"))
		err := blob.Write("hello", buf)
		assert.Nilf(t, err, "Error should be nil")
	})

	t.Run("ReadValid", func(t *testing.T) {
		val, err := blob.Read("hello")
		bytes, _ := ioutil.ReadAll(val)
		assert.Nilf(t, err, "Error should be nil")
		assert.Equal(t, string(bytes), "world", "Value should be world")
	})

	t.Run("ReadIncorrectNamespace", func(t *testing.T) {
		val, err := blob.Read("hello", store.BlobNamespace("bar"))
		assert.Equal(t, store.ErrNotFound, err, "Error should be not found")
		assert.Nil(t, val, "Value should be nil")
	})

	t.Run("ReadCorrectNamespace", func(t *testing.T) {
		val, err := blob.Read("hello", store.BlobNamespace("micro"))
		bytes, _ := ioutil.ReadAll(val)
		assert.Nil(t, err, "Error should be nil")
		assert.Equal(t, string(bytes), "world", "Value should be world")
	})

	t.Run("DeleteIncorrectNamespace", func(t *testing.T) {
		err := blob.Delete("hello", store.BlobNamespace("bar"))
		assert.Equal(t, store.ErrNotFound, err, "Error should be not found")
	})

	t.Run("DeleteCorrectNamespaceIncorrectKey", func(t *testing.T) {
		err := blob.Delete("world", store.BlobNamespace("micro"))
		assert.Equal(t, store.ErrNotFound, err, "Error should be not found")
	})

	t.Run("DeleteCorrectNamespace", func(t *testing.T) {
		err := blob.Delete("hello", store.BlobNamespace("micro"))
		assert.Nil(t, err, "Error should be nil")
	})

	t.Run("ReadDeletedKey", func(t *testing.T) {
		res, err := blob.Read("hello", store.BlobNamespace("micro"))
		assert.Equal(t, store.ErrNotFound, err, "Error should be not found")
		assert.Nil(t, res, "Result should be nil")
	})
}
