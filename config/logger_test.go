package config

import "testing"

func TestConfig_GetLogger_InvalidFormat(t *testing.T) {
	_, err := DefaultConfig().GetLogger(LogConfig{Level: "info", Format: "invalid"})
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
}

func TestConfig_GetLogger_InvalidLevel(t *testing.T) {
	_, err := DefaultConfig().GetLogger(LogConfig{Level: "invalid", Format: "json"})
	if err == nil {
		t.Fatal("expected error for invalid level")
	}
}
