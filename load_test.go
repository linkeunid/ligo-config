package ligo_config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_ReadsEnvFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("STANDALONE_KEY=loaded\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	svc, err := Load(WithEnvFiles(path))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if v, _ := svc.Get("STANDALONE_KEY"); v != "loaded" {
		t.Errorf("STANDALONE_KEY = %q, want loaded", v)
	}
}

func TestLoad_ValidatorFailureReturnsError(t *testing.T) {
	_, err := Load(
		WithIgnoreEnvFile(true),
		WithValidate(func(*Service) error { return errors.New("missing required") }),
	)
	if err == nil {
		t.Fatal("expected validator error")
	}
}

func TestMustLoad_PanicsOnError(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustLoad should panic on validator error")
		}
	}()
	_ = MustLoad(
		WithIgnoreEnvFile(true),
		WithValidate(func(*Service) error { return errors.New("boom") }),
	)
}
