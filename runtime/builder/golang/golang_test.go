package golang

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/micro/go-micro/v3/runtime/builder"
	"github.com/stretchr/testify/assert"
)

var (
	testMainGo   = "package main; import \"fmt\"; func main() { fmt.Println(\"HelloWorld\") }"
	testSecondGo = "package main; import \"fmt\"; func init() { fmt.Println(\"Init\") }"
)

func TestGolangBuilder(t *testing.T) {
	t.Run("NoArchive", func(t *testing.T) {
		buf := bytes.NewBuffer([]byte(testMainGo))
		err := testBuilder(t, buf)
		assert.Nil(t, err, "No error should be returned")
	})

	t.Run("InvalidArchive", func(t *testing.T) {
		buf := bytes.NewBuffer([]byte(testMainGo))
		err := testBuilder(t, buf, builder.Archive("foo"))
		assert.Error(t, err, "An error should be returned")
	})

	t.Run("TarArchive", func(t *testing.T) {
		// Create a tar writer
		tf := bytes.NewBuffer(nil)
		tw := tar.NewWriter(tf)

		// Add some files to the archive.
		var files = []struct {
			Name, Body string
		}{
			{"main.go", testMainGo},
			{"second.go", testSecondGo},
		}
		for _, file := range files {
			hdr := &tar.Header{
				Name: file.Name,
				Mode: 0600,
				Size: int64(len(file.Body)),
			}
			if err := tw.WriteHeader(hdr); err != nil {
				t.Fatal(err)
			}
			if _, err := tw.Write([]byte(file.Body)); err != nil {
				t.Fatal(err)
			}
		}
		if err := tw.Close(); err != nil {
			t.Fatal(err)
		}

		err := testBuilder(t, tf, builder.Archive("tar"))
		assert.Nil(t, err, "No error should be returned")
	})

	t.Run("ZipArchive", func(t *testing.T) {
		// Create a buffer to write our archive to.
		buf := new(bytes.Buffer)

		// Create a new zip archive.
		w := zip.NewWriter(buf)
		defer w.Close()

		// Add some files to the archive.
		var files = []struct {
			Name, Body string
		}{
			{"main.go", testMainGo},
			{"second.go", testSecondGo},
		}
		for _, file := range files {
			f, err := w.Create(file.Name)
			if err != nil {
				t.Fatal(err)
			}
			_, err = f.Write([]byte(file.Body))
			if err != nil {
				t.Fatal(err)
			}
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}

		err := testBuilder(t, buf, builder.Archive("zip"))
		assert.Nil(t, err, "No error should be returned")
	})
}

func testBuilder(t *testing.T, buf io.Reader, opts ...builder.Option) error {
	// setup the builder
	builder, err := NewBuilder()
	if err != nil {
		return fmt.Errorf("Error creating the builder: %v", err)
	}

	// build the source
	res, err := builder.Build(buf, opts...)
	if err != nil {
		return fmt.Errorf("Error building source: %v", err)
	}

	// write the binary to a tmp file and make it executable
	file, err := ioutil.TempFile(os.TempDir(), "res")
	if err != nil {
		return fmt.Errorf("Error creating tmp output file: %v", err)
	}
	if _, err := io.Copy(file, res); err != nil {
		return fmt.Errorf("Error copying binary to tmp file: %v", err)
	}
	os.Chmod(file.Name(), 0111)
	defer os.Remove(file.Name())

	// execute the binary
	cmd := exec.Command(file.Name())
	outp, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("Error executing binary: %v", err)
	}
	if !strings.Contains(string(outp), "HelloWorld") {
		return fmt.Errorf("Output does not contain HelloWorld")
	}
	// when an archive is used we also check for the second file to be loaded
	if len(opts) > 0 && !strings.Contains(string(outp), "Init") {
		return fmt.Errorf("Output does not contain Init")
	}

	return nil
}
