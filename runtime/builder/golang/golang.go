package golang

import (
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
	"github.com/micro/go-micro/v3/util/tar"
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
	case "zip":
		err = unarchiveZip(src, dir)
	case "tar":
		err = tar.Unarchive(src, dir)
	default:
		return nil, errors.New("Invalid Archive")
	}
	if err != nil {
		return nil, err
	}

	// build the binary
	cmd := exec.Command(g.cmdPath, "build", "-o", "micro_build", ".")
	cmd.Dir = filepath.Join(dir, options.Entrypoint)
	cmd.Env = append(os.Environ(),
		"GO111MODULE=auto",
		"CGO_ENABLED=0",
		"GOOS=linux",
		"GOARCH=amd64",
	)

	files, err := ioutil.ReadDir(cmd.Dir)
	if err != nil {
		fmt.Println("Err listing files in", cmd.Dir, err)
	}
	for _, f := range files {
		fmt.Println(f.Name())
	}

	var stdout, errout bytes.Buffer
	cmd.Stderr = &errout
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		fmt.Println(errout.String(), stdout.String())
		return nil, fmt.Errorf("Error building service: %v", errout.String())
	}

	// read the bytes from the file
	dst, err := ioutil.ReadFile(filepath.Join(dir, "micro_build"))
	if err != nil {
		return nil, err
	}
	fmt.Println("BUILD FINISHED OKAY", len(dst))

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
