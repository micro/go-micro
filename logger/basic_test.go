package logger

import (
	"errors"
	"fmt"
	"os"
	"testing"
)

func TestName(t *testing.T) {
	l := NewLogger()

	if l.String() != "basic" {
		t.Errorf("error: name expected 'basic' actual: %s", l.String())
	}

	t.Logf("testing logger name: %s", l.String())
}

func TestSetLevel(t *testing.T) {
	SetGlobalLogger(NewLogger())
	SetGlobalLevel(DebugLevel)
	Debugf("test show debug: %s", "debug msg")

	SetGlobalLevel(InfoLevel)
	Debugf("test non-show debug: %s", "debug msg")
}

func TestWithFields(t *testing.T) {
	l := NewLogger(WithFields(map[string]interface{}{
		"name":  "sumo",
		"age":   99,
		"alive": true,
	}))
	SetGlobalLogger(l)
	Info("test with fields")
	Infow("test with fields", map[string]interface{}{"weight": 3.14159265359, "name": "demo"})
}

func TestWithError(t *testing.T) {
	l := NewLogger(WithFields(map[string]interface{}{
		"name":  "sumo",
		"age":   99,
		"alive": true,
	}))
	SetGlobalLogger(l)
	Error("test with fields")
	Errorw("test with fields", fmt.Errorf("Error %v: %w", "nested", errors.New("root error message")))
}

func ExampleLog() {
	SetGlobalLogger(NewLogger(WithOutput(os.Stdout)))
	Info("test show info: ", "msg ", true, 45.65)
	Infof("test show infof: name: %s, age: %d", "sumo", 99)
	Infow("test show fields", map[string]interface{}{
		"name":  "sumo",
		"age":   99,
		"alive": true,
	})
	// Output:
	// {"message":"test show info: msg true 45.65"}
	// {"message":"test show infof: name: sumo, age: 99"}
	// {"age":99,"alive":true,"message":"test show fields","name":"sumo"}
}
