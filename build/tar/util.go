package tar

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func Compress(source, dest string) error {
	// tar + gzip
	var buf bytes.Buffer
	_ = compress(source, &buf)

	// write the .tar.gzip
	fileToWrite, err := os.OpenFile(dest, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return err
	}
	_, err = io.Copy(fileToWrite, &buf)
	return err
}

func compress(src string, buf io.Writer) error {
	// tar > gzip > buf
	zr := gzip.NewWriter(buf)
	tw := tar.NewWriter(zr)

	// walk through every file in the folder
	filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {

		// generate tar header
		header, err := tar.FileInfoHeader(fi, file)
		if err != nil {
			return err
		}

		// must provide real name
		// (see https://golang.org/src/archive/tar/common.go?#L626)

		srcWithSlash := src
		if !strings.HasSuffix(src, string(filepath.Separator)) {
			srcWithSlash = src + string(filepath.Separator)
		}
		header.Name = strings.ReplaceAll(file, srcWithSlash, "")
		if header.Name == src || len(strings.TrimSpace(header.Name)) == 0 {
			return nil
		}

		// @todo This is a quick hack to speed up whole repo uploads
		// https://github.com/micro/micro/pull/956
		if !fi.IsDir() && !strings.HasSuffix(header.Name, ".go") &&
			!strings.HasSuffix(header.Name, ".mod") &&
			!strings.HasSuffix(header.Name, ".sum") {
			return nil
		}

		// write header
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if fi.IsDir() {
			return nil
		}

		// if not a dir, write file content

		data, err := os.Open(file)
		if err != nil {
			return err
		}
		if _, err := io.Copy(tw, data); err != nil {
			return err
		}

		return nil
	})

	// produce tar
	if err := tw.Close(); err != nil {
		return err
	}
	// produce gzip
	if err := zr.Close(); err != nil {
		return err
	}
	//
	return nil
}
