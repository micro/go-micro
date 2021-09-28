package hystrix

import (
	"github.com/afex/hystrix-go/hystrix"
)

// CommandConfig is used to tune circuit settings at runtime
type CommandConfig struct {
	Timeout                int
	MaxConcurrentRequests  int
	RequestVolumeThreshold int
	SleepWindow            int
	ErrorPercentThreshold  int
}

// Configure applies settings for a set of circuits
func Configure(cmds map[string]CommandConfig) {
	for k, v := range cmds {
		ConfigureCommand(k, v)
	}
}

// ConfigureCommand applies settings for a circuit
func ConfigureCommand(name string, config CommandConfig) {
	hystrix.ConfigureCommand(name, hystrix.CommandConfig{
		Timeout:                config.Timeout,
		MaxConcurrentRequests:  config.MaxConcurrentRequests,
		RequestVolumeThreshold: config.RequestVolumeThreshold,
		SleepWindow:            config.SleepWindow,
		ErrorPercentThreshold:  config.ErrorPercentThreshold,
	})
}

// ConfigureDefault applies default settings for all circuits
func ConfigureDefault(config CommandConfig) {
	if config.Timeout != 0 {
		hystrix.DefaultTimeout = config.Timeout
	}
	if config.MaxConcurrentRequests != 0 {
		hystrix.DefaultMaxConcurrent = config.MaxConcurrentRequests
	}
	if config.RequestVolumeThreshold != 0 {
		hystrix.DefaultVolumeThreshold = config.RequestVolumeThreshold
	}
	if config.SleepWindow != 0 {
		hystrix.DefaultSleepWindow = config.SleepWindow
	}
	if config.ErrorPercentThreshold != 0 {
		hystrix.DefaultErrorPercentThreshold = config.ErrorPercentThreshold
	}
}
