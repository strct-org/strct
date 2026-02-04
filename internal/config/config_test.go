package config

import (
	"os"
	"testing"
)

func TestGetEnvAsInt(t *testing.T) {
	tests := []struct {
		name     string
		envKey   string
		envVal   string
		fallback int
		want     int
	}{
		{"Valid Integer", "TEST_PORT", "9090", 8080, 9090},
		{"Empty String", "TEST_EMPTY", "", 8080, 8080},
		{"Invalid String", "TEST_BAD", "abc", 7000, 7000},
		{"Unset Variable", "TEST_UNSET", "", 5000, 5000}, 
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name != "Unset Variable" {
				t.Setenv(tt.envKey, tt.envVal)
			} else {
				os.Unsetenv(tt.envKey)
			}

			got := getEnvAsInt(tt.envKey, tt.fallback)

			if got != tt.want {
				t.Errorf("getEnvAsInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

