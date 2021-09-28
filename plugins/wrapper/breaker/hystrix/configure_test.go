package hystrix

import (
	"testing"
	"time"

	"github.com/afex/hystrix-go/hystrix"
)

func TestConfigure(t *testing.T) {
	command, timeout := "testing.configure", 200
	Configure(map[string]CommandConfig{command: {Timeout: timeout}})
	configures := hystrix.GetCircuitSettings()
	if c, ok := configures[command]; !ok || c.Timeout != time.Duration(timeout)*time.Millisecond {
		t.Fail()
	}
}

func TestConfigureCommand(t *testing.T) {
	command, timeout := "testing.configureCommand", 300
	ConfigureCommand(command, CommandConfig{Timeout: timeout})
	configures := hystrix.GetCircuitSettings()
	if c, ok := configures[command]; !ok || c.Timeout != time.Duration(timeout)*time.Millisecond {
		t.Fail()
	}
}

func TestConfigureDefault(t *testing.T) {
	timeout, maxConcurrent, reqThreshold, sleepWindow, errThreshold := 100, 20, 10, 500, 5
	ConfigureDefault(CommandConfig{
		Timeout:                timeout,
		MaxConcurrentRequests:  maxConcurrent,
		RequestVolumeThreshold: reqThreshold,
		SleepWindow:            sleepWindow,
		ErrorPercentThreshold:  errThreshold})
	if hystrix.DefaultTimeout != timeout {
		t.Fail()
	}
	if hystrix.DefaultVolumeThreshold != reqThreshold {
		t.Fail()
	}
	if hystrix.DefaultMaxConcurrent != maxConcurrent {
		t.Fail()
	}
	if hystrix.DefaultSleepWindow != sleepWindow {
		t.Fail()
	}
	if hystrix.DefaultErrorPercentThreshold != errThreshold {
		t.Fail()
	}
}
