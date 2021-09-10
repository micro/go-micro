package generator

import (
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// Generator is the interface that generates project template files.
//
// Generate accepts a list of files and generates them based on their template.
type Generator interface {
	Generate([]File) error
}

type generator struct {
	opts Options
}

// File represents a file to generate.
type File struct {
	// Path specifies where the file will reside.
	Path string
	// Template is the template used to generate the file.
	Template string
}

// Generate generates project template files.
func (g *generator) Generate(files []File) error {
	for _, file := range files {
		fp := filepath.Join(g.opts.Directory, file.Path)
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

		err = t.Execute(f, g.opts)
		if err != nil {
			return err
		}
	}

	return nil
}

// New returns a new generator struct.
func New(opts ...Option) Generator {
	var options Options
	for _, o := range opts {
		o(&options)
	}

	return &generator{
		opts: options,
	}
}
