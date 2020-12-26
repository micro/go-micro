package logrus

import (
	"errors"
	"os"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/micro/go-micro/v2/logger"
)

func TestName(t *testing.T) {
	l := NewLogger()

	if l.String() != "logrus" {
		t.Errorf("error: name expected 'logrus' actual: %s", l.String())
	}

	t.Logf("testing logger name: %s", l.String())
}

func TestWithFields(t *testing.T) {
	l := NewLogger(logger.WithOutput(os.Stdout)).Fields(map[string]interface{}{
		"k1": "v1",
		"k2": 123456,
	})

	logger.DefaultLogger = l

	logger.Log(logger.InfoLevel, "testing: Info")
	logger.Logf(logger.InfoLevel, "testing: %s", "Infof")
}

func TestWithError(t *testing.T) {
	l := NewLogger().Fields(map[string]interface{}{"error": errors.New("boom!")})
	logger.DefaultLogger = l

	logger.Log(logger.InfoLevel, "testing: error")
}

func TestWithLogger(t *testing.T) {
	// with *logrus.Logger
	l := NewLogger(WithLogger(logrus.StandardLogger())).Fields(map[string]interface{}{
		"k1": "v1",
		"k2": 123456,
	})
	logger.DefaultLogger = l
	logger.Log(logger.InfoLevel, "testing: with *logrus.Logger")

	// with *logrus.Entry
	el := NewLogger(WithLogger(logrus.NewEntry(logrus.StandardLogger()))).Fields(map[string]interface{}{
		"k3": 3.456,
		"k4": true,
	})
	logger.DefaultLogger = el
	logger.Log(logger.InfoLevel, "testing: with *logrus.Entry")
}

func TestJSON(t *testing.T) {
	logger.DefaultLogger = NewLogger(WithJSONFormatter(&logrus.JSONFormatter{}))

	logger.Logf(logger.InfoLevel, "test logf: %s", "name")
}

func TestSetLevel(t *testing.T) {
	logger.DefaultLogger = NewLogger()

	logger.Init(logger.WithLevel(logger.DebugLevel))
	logger.Logf(logger.DebugLevel, "test show debug: %s", "debug msg")

	logger.Init(logger.WithLevel(logger.InfoLevel))
	logger.Logf(logger.DebugLevel, "test non-show debug: %s", "debug msg")
}

func TestWithReportCaller(t *testing.T) {
	logger.DefaultLogger = NewLogger(ReportCaller())

	logger.Logf(logger.InfoLevel, "testing: %s", "WithReportCaller")
}
