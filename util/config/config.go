package config

import (
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	conf "github.com/micro/go-micro/v2/config"
	"github.com/micro/go-micro/v2/config/source/file"
)

// FileName for global micro config
const FileName = ".micro"

// Get a value from the .micro file
func Get(key string) (string, error) {
	// get the filepath
	fp, err := filePath()
	if err != nil {
		return "", err
	}

	// create a new config
	c, err := conf.NewConfig(
		conf.WithSource(
			file.NewSource(
				file.WithPath(fp),
			),
		),
	)
	if err != nil {
		return "", err
	}

	// load the config
	if err := c.Load(); err != nil {
		return "", err
	}

	// set a value
	tk := c.Get(key).String("")

	return strings.TrimSpace(tk), nil
}

// Set a value in the .micro file
func Set(key, value string) error {
	// get the filepath
	fp, err := filePath()
	if err != nil {
		return err
	}

	// write the file if it does not exist
	if _, err := os.Stat(fp); os.IsNotExist(err) {
		ioutil.WriteFile(fp, []byte{}, 0644)
	} else if err != nil {
		return err
	}

	// create a new config
	c, err := conf.NewConfig(
		conf.WithSource(
			file.NewSource(
				file.WithPath(fp),
			),
		),
	)
	if err != nil {
		return err
	}

	// load the config
	if err := c.Load(); err != nil {
		return err
	}

	// set a value
	c.Set(value, key)

	// write the file
	return ioutil.WriteFile(fp, c.Bytes(), 0644)
}

func filePath() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join(usr.HomeDir, FileName), nil
}
