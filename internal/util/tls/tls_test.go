package tls

import (
	"os"
	"testing"
)

func TestConfig(t *testing.T) {
	tests := []struct {
		name         string
		envVar       string
		envValue     string
		wantInsecure bool
		description  string
	}{
		{
			name:         "secure_by_default",
			envVar:       "",
			envValue:     "",
			wantInsecure: false,
			description:  "v6: certificate verification is on by default",
		},
		{
			name:         "insecure_opt_out",
			envVar:       "MICRO_TLS_INSECURE",
			envValue:     "true",
			wantInsecure: true,
			description:  "MICRO_TLS_INSECURE=true skips verification (dev/self-signed)",
		},
		{
			name:         "insecure_disabled_stays_secure",
			envVar:       "MICRO_TLS_INSECURE",
			envValue:     "false",
			wantInsecure: false,
			description:  "MICRO_TLS_INSECURE=false stays secure",
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
