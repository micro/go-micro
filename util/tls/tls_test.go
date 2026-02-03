package tls

import (
	"os"
	"testing"
)

func TestConfig(t *testing.T) {
	tests := []struct {
		name           string
		envVar         string
		envValue       string
		wantInsecure   bool
		description    string
	}{
		{
			name:         "default_insecure_for_backward_compatibility",
			envVar:       "",
			envValue:     "",
			wantInsecure: true,
			description:  "Default should remain insecure for backward compatibility (will change in v6)",
		},
		{
			name:         "secure_mode_enabled",
			envVar:       "MICRO_TLS_SECURE",
			envValue:     "true",
			wantInsecure: false,
			description:  "MICRO_TLS_SECURE=true should enable certificate verification",
		},
		{
			name:         "secure_mode_disabled",
			envVar:       "MICRO_TLS_SECURE",
			envValue:     "false",
			wantInsecure: true,
			description:  "MICRO_TLS_SECURE=false should remain insecure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up environment
			os.Unsetenv("MICRO_TLS_SECURE")
			os.Unsetenv("MICRO_TLS_INSECURE")
			// Suppress warning in tests
			os.Setenv("IN_TRAVIS_CI", "yes")
			defer os.Unsetenv("IN_TRAVIS_CI")

			// Set environment variable if specified
			if tt.envVar != "" {
				os.Setenv(tt.envVar, tt.envValue)
				defer os.Unsetenv(tt.envVar)
			}

			config := Config()

			if config == nil {
				t.Fatal("Config() returned nil")
			}

			if config.InsecureSkipVerify != tt.wantInsecure {
				t.Errorf("%s: InsecureSkipVerify = %v, want %v",
					tt.description, config.InsecureSkipVerify, tt.wantInsecure)
			}

			// Verify MinVersion is set correctly
			if config.MinVersion == 0 {
				t.Error("MinVersion should be set")
			}
		})
	}
}

func TestSecureConfig(t *testing.T) {
	config := SecureConfig()

	if config == nil {
		t.Fatal("SecureConfig() returned nil")
	}

	if config.InsecureSkipVerify {
		t.Error("SecureConfig should have InsecureSkipVerify set to false")
	}

	if config.MinVersion == 0 {
		t.Error("MinVersion should be set")
	}
}

func TestInsecureConfig(t *testing.T) {
	config := InsecureConfig()

	if config == nil {
		t.Fatal("InsecureConfig() returned nil")
	}

	if !config.InsecureSkipVerify {
		t.Error("InsecureConfig should have InsecureSkipVerify set to true")
	}

	if config.MinVersion == 0 {
		t.Error("MinVersion should be set")
	}
}
