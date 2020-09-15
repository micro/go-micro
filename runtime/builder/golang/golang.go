package golang

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/micro/go-micro/v3/runtime/builder"
	"github.com/micro/go-micro/v3/runtime/local"
)

// NewBuilder returns a golang builder which can build a go binary given some source
func NewBuilder() (builder.Builder, error) {
	path, err := locateGo()
	if err != nil {
		return nil, fmt.Errorf("Error locating go binary: %v", err)
	}

	return &golang{
		cmdPath: path,
		tmpDir:  os.TempDir(),
	}, nil
}

type golang struct {
	cmdPath string
	tmpDir  string
}

// Build a binary using source. If an archive was used, e.g. tar, this should be specified in the
// options. If no archive option is passed, the builder will treat the source as a single file.
func (g *golang) Build(src io.Reader, opts ...builder.Option) (io.Reader, error) {
	// parse the options
	var options builder.Options
	for _, o := range opts {
		o(&options)
	}

	// create a tmp dir to contain the source
	dir, err := ioutil.TempDir(g.tmpDir, "src")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	// decode the source and write to the tmp directory
	switch options.Archive {
	case "":
		err = writeFile(src, dir)
	case "tar":
		err = unarchiveTar(src, dir)
	case "zip":
		err = unarchiveZip(src, dir)
	default:
		return nil, errors.New("Invalid Archive")
	}
	if err != nil {
		return nil, err
	}

	// determine the entrypoint if one wasn't provided
	if len(options.Entrypoint) == 0 {
		ep, err := local.Entrypoint(dir)
		if err != nil {
			return nil, err
		}
		options.Entrypoint = ep
	}

	// create a file for the output to be written to
	out, err := ioutil.TempFile(dir, "output")
	if err != nil {
		return nil, err
	}
	defer out.Close()

	// build the binary
	cmd := exec.Command(g.cmdPath, "build", "-o", out.Name(), options.Entrypoint)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	// read the bytes from the file
	dst, err := ioutil.ReadFile(out.Name())
	if err != nil {
		return nil, err
	}

	return bytes.NewBuffer(dst), nil
}

// writeFile takes a single file to a directory
func writeFile(src io.Reader, dir string) error {
	// copy the source to the temp file
	bytes, err := ioutil.ReadAll(src)
	if err != nil {
		return err
	}

	// write the file, note: in order for the golang builder to access the file, it cannot be
	// os.ModeTemp. This is okay because we delete all the files in the tmp dir at the end of this
	// function.
	return ioutil.WriteFile(filepath.Join(dir, "main.go"), bytes, os.ModePerm)
}

// unarchiveTar decodes the source in a tar and writes it to a directory
func unarchiveTar(src io.Reader, dir string) error {
	tr := tar.NewReader(src)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		path := filepath.Join(dir, hdr.Name)
		bytes, err := ioutil.ReadAll(tr)
		if err != nil {
			return err
		}

		if err := ioutil.WriteFile(path, bytes, os.ModePerm); err != nil {
			return err
		}
	}
	return nil
}

// unarchiveZip decodes the source in a zip and writes it to a directory
func unarchiveZip(src io.Reader, dir string) error {
	// create a new buffer with the source, this is required because zip.NewReader takes a io.ReaderAt
	// and not an io.Reader
	buff := bytes.NewBuffer([]byte{})
	size, err := io.Copy(buff, src)
	if err != nil {
		return err
	}

	// create the zip
	reader := bytes.NewReader(buff.Bytes())
	zip, err := zip.NewReader(reader, size)
	if err != nil {
		return err
	}

	// write the files in the zip to our tmp dir
	for _, f := range zip.File {
		rc, err := f.Open()
		if err != nil {
			return err
		}

		bytes, err := ioutil.ReadAll(rc)
		if err != nil {
			return err
		}

		path := filepath.Join(dir, f.Name)
		if err := ioutil.WriteFile(path, bytes, os.ModePerm); err != nil {
			return err
		}

		if err := rc.Close(); err != nil {
			return err
		}
	}

	return nil
}

// locateGo locates the go command
func locateGo() (string, error) {
	if gr := os.Getenv("GOROOT"); len(gr) > 0 {
		return filepath.Join(gr, "bin", "go"), nil
	}

	// check path
	for _, p := range filepath.SplitList(os.Getenv("PATH")) {
		bin := filepath.Join(p, "go")
		if _, err := os.Stat(bin); err == nil {
			return bin, nil
		}
	}

	return exec.LookPath("go")
}
