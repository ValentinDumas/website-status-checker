package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetConfigPath(t *testing.T) {
	path, err := GetConfigPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasSuffix(path, "sites.yaml") {
		t.Errorf("expected path to end with sites.yaml, got %q", path)
	}
	if !strings.Contains(path, "WebsiteStatusChecker") {
		t.Errorf("expected path to contain WebsiteStatusChecker, got %q", path)
	}
}

func TestEnsureConfigExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "WebsiteStatusChecker", "sites.yaml")

	// 1. Should create file on first run
	err := EnsureConfigExists(path)
	if err != nil {
		t.Fatalf("EnsureConfigExists failed on first run: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read created config: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected default YAML to be written, got empty file")
	}

	// 2. Should not overwrite an existing file
	customData := []byte("custom configuration")
	err = os.WriteFile(path, customData, 0644)
	if err != nil {
		t.Fatalf("failed to write custom data: %v", err)
	}

	err = EnsureConfigExists(path)
	if err != nil {
		t.Fatalf("EnsureConfigExists failed on second run: %v", err)
	}

	dataAfter, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read config after second run: %v", err)
	}

	if string(dataAfter) != string(customData) {
		t.Errorf("EnsureConfigExists overwrote existing file")
	}
}
