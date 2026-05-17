package ligo_config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestModule_ReturnsLigoModule(t *testing.T) {
	m := Module()
	if m.Name != ModuleName {
		t.Errorf("Module name = %q, want %q", m.Name, ModuleName)
	}
}

func TestModule_BootstrapInitLoadsFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("HELLO=world\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	opts := defaultOptions()
	opts.envFiles = []string{path}
	svc := newService()
	b := &configBootstrap{svc: svc, opts: opts}

	if err := b.init(); err != nil {
		t.Fatalf("init: %v", err)
	}
	if v, _ := svc.Get("HELLO"); v != "world" {
		t.Errorf("HELLO = %q, want world", v)
	}
}

func TestModule_ValidatorErrorAbortsInit(t *testing.T) {
	opts := defaultOptions()
	opts.ignoreEnvFile = true
	opts.validate = func(*Service) error {
		return errors.New("config invalid")
	}

	b := &configBootstrap{svc: newService(), opts: opts}
	err := b.init()
	if err == nil {
		t.Fatal("expected validator error")
	}
	if err.Error() != "config invalid" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestModule_ValidatorReceivesLoadedService(t *testing.T) {
	t.Setenv("VALIDATE_KEY", "ok")

	captured := ""
	opts := defaultOptions()
	opts.ignoreEnvFile = true
	opts.validate = func(s *Service) error {
		v, _ := s.Get("VALIDATE_KEY")
		captured = v
		return nil
	}

	b := &configBootstrap{svc: newService(), opts: opts}
	if err := b.init(); err != nil {
		t.Fatalf("init: %v", err)
	}
	if captured != "ok" {
		t.Errorf("validator saw %q, want ok", captured)
	}
}
