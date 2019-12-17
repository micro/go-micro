package kubernetes

import "github.com/micro/go-micro/debug/log"

import (
	"encoding/json"
	"fmt"
	"os"
)

func write(l log.Record) {
	if m, err := json.Marshal(l); err == nil {
		fmt.Fprintf(os.Stderr, "%s", m)
	}
}
