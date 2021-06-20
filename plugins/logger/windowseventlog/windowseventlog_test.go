// +build windows

package windowseventlog

import (
	"testing"

	"github.com/asim/go-micro/v3/logger"
	"github.com/stretchr/testify/assert"
)

func TestWindowsEventLog(t *testing.T) {

	var l logger.Logger = NewLogger(WithSrc("windows logger test"))

	l.Log(logger.TraceLevel, "test trace level")
	l.Log(logger.DebugLevel, "test debug")
	l.Log(logger.InfoLevel, "test info level")
	l.Log(logger.WarnLevel, "test warn level")
	l.Log(logger.ErrorLevel, "test error level")
	l.Log(logger.FatalLevel, "test fatal level")

	l.Logf(logger.TraceLevel, "%s trace level", "test formatted")
	l.Logf(logger.DebugLevel, "%s debug", "test formatted")
	l.Logf(logger.InfoLevel, "%s info level", "test formatted")
	l.Logf(logger.WarnLevel, "%s warn level", "test formatted")
	l.Logf(logger.ErrorLevel, "%s error level", "test formatted")
	l.Logf(logger.FatalLevel, "%s fatal level", "test formatted")

	assert.Equal(t, l.String(), "windowseventlog")

}
