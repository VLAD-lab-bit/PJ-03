package config

import (
	"testing"
)

func TestLoadConfig(t *testing.T) {
	config, err := LoadConfig("../../config.json")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if len(config.RSS) == 0 {
		t.Error("RSS feed list should not be empty")
	}

	if config.RequestPeriod <= 0 {
		t.Error("Request period should be greater than 0")
	}
}
