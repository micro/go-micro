// Package new generates micro service templates
package new

import (
	"fmt"
	"go/build"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/xlab/treeprint"
	tmpl "go-micro.dev/v5/cmd/micro/cli/new/template"
)

func protoComments(goDir, alias string) []string {
	return []string{
		"\ndownload protoc zip packages (protoc-$VERSION-$PLATFORM.zip) and install:\n",
		"visit https://github.com/protocolbuffers/protobuf/releases",
		"\ncompile the proto file " + alias + ".proto:\n",
		"cd " + alias,
		"go mod tidy",
		"make proto\n",
	}
}

type config struct {
	// foo
	Alias string
	// github.com/micro/foo
	Dir string
	// $GOPATH/src/github.com/micro/foo
	GoDir string
	// $GOPATH
	GoPath string
	// UseGoPath
	UseGoPath bool
	// Files
	Files []file
	// Comments
	Comments []string
}

type file struct {
	Path string
	Tmpl string
}

func write(c config, file, tmpl string) error {
	fn := template.FuncMap{
		"title": func(s string) string {
			return strings.ReplaceAll(strings.Title(s), "-", "")
		},
		"dehyphen": func(s string) string {
			return strings.ReplaceAll(s, "-", "")
		},
		"lower": func(s string) string {
			return strings.ToLower(s)
		},
	}

	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()

	t, err := template.New("f").Funcs(fn).Parse(tmpl)
	if err != nil {
		return err
	}

	return t.Execute(f, c)
}

func create(c config) error {
	// check if dir exists
	if _, err := os.Stat(c.Dir); !os.IsNotExist(err) {
		return fmt.Errorf("%s already exists", c.Dir)
	}

	fmt.Printf("Creating service %s\n\n", c.Alias)

	t := treeprint.New()

	// write the files
	for _, file := range c.Files {
		f := filepath.Join(c.Dir, file.Path)
		dir := filepath.Dir(f)

		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
		}

		addFileToTree(t, file.Path)
		if err := write(c, f, file.Tmpl); err != nil {
			return err
		}
	}

	// print tree
	fmt.Println(t.String())

	for _, comment := range c.Comments {
		fmt.Println(comment)
	}

	// just wait
	<-time.After(time.Millisecond * 250)

	return nil
}

func addFileToTree(root treeprint.Tree, file string) {
	split := strings.Split(file, "/")
	curr := root
	for i := 0; i < len(split)-1; i++ {
		n := curr.FindByValue(split[i])
		if n != nil {
			curr = n
		} else {
			curr = curr.AddBranch(split[i])
		}
	}
	if curr.FindByValue(split[len(split)-1]) == nil {
		curr.AddNode(split[len(split)-1])
	}
}

func Run(ctx *cli.Context) error {
	dir := ctx.Args().First()
	if len(dir) == 0 {
		fmt.Println("specify service name")
		return nil
	}

	// check if the path is absolute, we don't want this
	// we want to a relative path so we can install in GOPATH
	if path.IsAbs(dir) {
		fmt.Println("require relative path as service will be installed in GOPATH")
		return nil
	}

	// Check for protoc
	if _, err := exec.LookPath("protoc"); err != nil {
		fmt.Println("WARNING: protoc is not installed or not in your PATH.")
		fmt.Println("Please install protoc from https://github.com/protocolbuffers/protobuf/releases")
		fmt.Println("After installing, re-run 'make proto' in your service directory if needed.")
	}

	var goPath string
	var goDir string

	goPath = build.Default.GOPATH

	// don't know GOPATH, runaway....
	if len(goPath) == 0 {
		fmt.Println("unknown GOPATH")
		return nil
	}

	// attempt to split path if not windows
	if runtime.GOOS == "windows" {
		goPath = strings.Split(goPath, ";")[0]
	} else {
		goPath = strings.Split(goPath, ":")[0]
	}
	goDir = filepath.Join(goPath, "src", path.Clean(dir))

	c := config{
		Alias:     dir,
		Comments:  nil, // Remove redundant protoComments
		Dir:       dir,
		GoDir:     goDir,
		GoPath:    goPath,
		UseGoPath: false,
		Files: []file{
			{"main.go", tmpl.MainSRV},
			{"handler/" + dir + ".go", tmpl.HandlerSRV},
			{"proto/" + dir + ".proto", tmpl.ProtoSRV},
			{"Makefile", tmpl.Makefile},
			{"README.md", tmpl.Readme},
			{".gitignore", tmpl.GitIgnore},
		},
	}

	// set gomodule
	if os.Getenv("GO111MODULE") != "off" {
		c.Files = append(c.Files, file{"go.mod", tmpl.Module})
	}

	// create the files
	if err := create(c); err != nil {
		return err
	}

	// Run go mod tidy and make proto
	fmt.Println("\nRunning 'go mod tidy' and 'make proto'...")
	if err := runInDir(dir, "go mod tidy"); err != nil {
		fmt.Printf("Error running 'go mod tidy': %v\n", err)
	}
	if err := runInDir(dir, "make proto"); err != nil {
		fmt.Printf("Error running 'make proto': %v\n", err)
	}

	// Print updated tree including generated files
	fmt.Println("\nProject structure after 'make proto':")
	printTree(dir)

	fmt.Println("\nService created successfully! Start coding in your new service directory.")
	return nil
}

func runInDir(dir, cmd string) error {
	parts := strings.Fields(cmd)
	c := exec.Command(parts[0], parts[1:]...)
	c.Dir = dir
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func printTree(dir string) {
	t := treeprint.New()
	walk := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		if rel == "." {
			return nil
		}
		parts := strings.Split(rel, string(os.PathSeparator))
		curr := t
		for i := 0; i < len(parts)-1; i++ {
			n := curr.FindByValue(parts[i])
			if n != nil {
				curr = n
			} else {
				curr = curr.AddBranch(parts[i])
			}
		}
		if !info.IsDir() {
			curr.AddNode(parts[len(parts)-1])
		}
		return nil
	}
	filepath.Walk(dir, walk)
	fmt.Println(t.String())
}
