package basic_test

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/micro/go-micro/v2/logger"
	"github.com/micro/go-micro/v2/logger/basic"
	"github.com/micro/go-micro/v2/logger/log"
)

func TestName(t *testing.T) {
	l := basic.NewLogger()

	if l.String() != "basic" {
		t.Errorf("error: name expected 'basic' actual: %s", l.String())
	}

	t.Logf("testing logger name: %s", l.String())
}

func TestSetLevel(t *testing.T) {
	log.SetGlobalLevel(logger.DebugLevel)
	log.Debugf("test show debug: %s", "debug msg")

	log.SetGlobalLevel(logger.InfoLevel)
	log.Debugf("test non-show debug: %s", "debug msg")
}

func TestWithFields(t *testing.T) {
	l := basic.NewLogger(basic.WithFields(map[string]interface{}{
		"name":  "sumo",
		"age":   99,
		"alive": true,
	}))
	log.SetGlobalLogger(l)
	log.Info("test with fields")
	log.Infow("test with fields", map[string]interface{}{"weight": 3.14159265359, "name": "demo"})
}

func TestWithError(t *testing.T) {
	l := basic.NewLogger(basic.WithFields(map[string]interface{}{
		"name":  "sumo",
		"age":   99,
		"alive": true,
	}))
	log.SetGlobalLogger(l)
	log.Error("test with fields")
	log.Errorw("test with fields", fmt.Errorf("Error %v: %w", "nested", errors.New("root error message")))
}

func ExampleLog() {
	l := basic.NewLogger(basic.WithOutput(os.Stdout))
	log.SetGlobalLogger(l)
	log.Info("test show info: ", "msg ", true, 45.65)
	log.Infof("test show infof: name: %s, age: %d", "sumo", 99)
	log.Infow("test show fields", map[string]interface{}{
		"name":  "sumo",
		"age":   99,
		"alive": true,
	})
	// Output:
	// {"message":"test show info: msg true 45.65"}
	// {"message":"test show infof: name: sumo, age: 99"}
	// {"age":99,"alive":true,"message":"test show fields","name":"sumo"}
}
