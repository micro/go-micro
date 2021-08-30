package run

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/micro/cli/v2"
)

// NewCommand returns a new run command.
func NewCommand() *cli.Command {
	return &cli.Command{
		Name:   "run",
		Usage:  "Build and run a service continuously",
		Action: Run,
	}
}

// Run runs a service and watches the project directory for change events. On
// write, the service is restarted. Exits on error.
func Run(ctx *cli.Context) error {
	service := newService()
	if err := service.Start(); err != nil {
		return err
	}
	defer service.Stop()

	sig := make(chan os.Signal)
	done := make(chan bool)
	signal.Notify(sig, os.Interrupt)
	go func() {
		<-sig
		service.Stop()
		done <- true
	}()

	go func() {
		for {
			service.Wait()
			fmt.Println("Service has stopped, restarting service")
			service.Start()
			time.Sleep(time.Second)
		}
	}()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Println(err)
	}
	defer watcher.Close()

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					service.Stop()
				}
				if event.Op&fsnotify.Create == fsnotify.Create {
					watcher.Add(event.Name)
				}
				if event.Op&fsnotify.Remove == fsnotify.Remove {
					watcher.Remove(event.Name)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				fmt.Println("ERROR", err)
			}
		}
	}()

	var files []string
	filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		files = append(files, path)
		return nil
	})

	for _, file := range files {
		if err := watcher.Add(file); err != nil {
			return err
		}
	}

	<-done
	return nil
}
