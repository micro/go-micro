package generate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type Config struct {
	Service  string
	Vendor   string
	Dir      string
	Comments []string
	Client   bool
	Jaeger   bool
	Skaffold bool
}

type File struct {
	Path     string
	Template string
}

func Create(files []File, c Config) error {
	for _, file := range files {
		fp := filepath.Join(c.Dir, file.Path)
		dir := filepath.Dir(fp)

		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
		}

		f, err := os.Create(fp)
		if err != nil {
			return err
		}

		fn := template.FuncMap{
			"dehyphen": func(s string) string {
				return strings.ReplaceAll(s, "-", "")
			},
			"lower": strings.ToLower,
			"title": func(s string) string {
				return strings.ReplaceAll(strings.Title(s), "-", "")
			},
		}
		t, err := template.New(fp).Funcs(fn).Parse(file.Template)
		if err != nil {
			return err
		}

		err = t.Execute(f, c)
		if err != nil {
			return err
		}
	}

	for _, comment := range c.Comments {
		fmt.Println(comment)
	}

	return nil
}
