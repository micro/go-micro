package apex

import (
	"testing"

	log "github.com/micro/go-micro/v2/logger"
)

var (
	l = New()
)

func TestName(t *testing.T) {
	l2 := New(WithTextHandler())
	if l2.String() != "apex" {
		t.Errorf("name is error %s", l2.String())
	}
	t.Logf("test logger name: %s", l2.String())
}

func testLog(l log.Logger) {
	l.Logf(log.InfoLevel, "Test Logf with level: %s", "info")
	l.Logf(log.DebugLevel, "Test Logf with level: %s", "debug")
	l.Logf(log.ErrorLevel, "Test Logf with level: %s", "error")
	l.Logf(log.TraceLevel, "Test Logf with level: %s", "trace")
	l.Logf(log.WarnLevel, "Test Logf with level: %s", "warn")
	//l.Logf(log.FatalLevel, "Test Logf with level: %s", "fatal")
}

func TestJSON(t *testing.T) {
	l2 := New(WithJSONHandler(), WithLevel(log.TraceLevel)).Fields(map[string]interface{}{
		"Format": "JSON",
	})
	testLog(l2)
}

func TestText(t *testing.T) {
	l2 := New(WithTextHandler(), WithLevel(log.TraceLevel)).Fields(map[string]interface{}{
		"Format": "Text",
	})
	testLog(l2)
}

func TestCLI(t *testing.T) {
	l2 := New(WithCLIHandler(), WithLevel(log.TraceLevel)).Fields(map[string]interface{}{
		"Format": "CLI",
	})
	testLog(l2)
}

func TestWithLevel(t *testing.T) {
	l2 := New(WithTextHandler(), WithLevel(log.DebugLevel))
	l2.Logf(log.DebugLevel, "test show debug: %s", "debug msg")

	l3 := New(WithTextHandler(), WithLevel(log.InfoLevel))
	l3.Logf(log.DebugLevel, "test non-show debug: %s", "debug msg")
}

func TestWithFields(t *testing.T) {
	l2 := New(WithTextHandler()).Fields(map[string]interface{}{
		"k1": "v1",
		"k2": 123456,
	})
	l2.Logf(log.InfoLevel, "Testing with values")
}
