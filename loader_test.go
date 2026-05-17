package ligo_config

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadSources_EnvFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("FOO=bar\nBAZ=qux\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	opts := defaultOptions()
	opts.envFiles = []string{path}
	merged, err := loadSources(t.Context(), opts)
	if err != nil {
		t.Fatalf("loadSources: %v", err)
	}
	if merged["FOO"] != "bar" || merged["BAZ"] != "qux" {
		t.Errorf("got %v, want FOO=bar BAZ=qux", merged)
	}
}

func TestLoadSources_MissingEnvFileIsNotError(t *testing.T) {
	opts := defaultOptions()
	opts.envFiles = []string{"/does/not/exist/.env"}
	if _, err := loadSources(t.Context(), opts); err != nil {
		t.Errorf("missing env file should not error, got %v", err)
	}
}

func TestLoadSources_IgnoreEnvFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("SHOULD_NOT_APPEAR=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	opts := defaultOptions()
	opts.envFiles = []string{path}
	opts.ignoreEnvFile = true
	merged, err := loadSources(t.Context(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := merged["SHOULD_NOT_APPEAR"]; ok {
		t.Error(".env should be ignored when ignoreEnvFile is true")
	}
}

func TestLoadSources_ProcessEnvOverridesEnvFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("KEY_FROM_TEST=file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KEY_FROM_TEST", "env")

	opts := defaultOptions()
	opts.envFiles = []string{path}
	merged, err := loadSources(t.Context(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if merged["KEY_FROM_TEST"] != "env" {
		t.Errorf("process env should win over .env, got %q", merged["KEY_FROM_TEST"])
	}
}

func TestLoadSources_LoadersWinFlipsPrecedence(t *testing.T) {
	t.Setenv("LOADER_KEY", "from-env")
	opts := defaultOptions()
	opts.ignoreEnvFile = true
	opts.loaders = []Loader{
		func(context.Context) (map[string]string, error) {
			return map[string]string{"LOADER_KEY": "from-loader"}, nil
		},
	}

	// Default: process env wins.
	merged, _ := loadSources(t.Context(), opts)
	if merged["LOADER_KEY"] != "from-env" {
		t.Errorf("default: process env should win, got %q", merged["LOADER_KEY"])
	}

	// With WithLoadersWin: loaders override process env.
	opts.loadersWin = true
	merged, _ = loadSources(t.Context(), opts)
	if merged["LOADER_KEY"] != "from-loader" {
		t.Errorf("loadersWin: loader should override env, got %q", merged["LOADER_KEY"])
	}
}

func TestLoadSources_LoaderError(t *testing.T) {
	opts := defaultOptions()
	opts.ignoreEnvFile = true
	opts.loaders = []Loader{
		func(context.Context) (map[string]string, error) {
			return nil, errors.New("boom")
		},
	}
	_, err := loadSources(t.Context(), opts)
	if !errors.Is(err, ErrLoadFailed) {
		t.Errorf("loader error should wrap ErrLoadFailed, got %v", err)
	}
}

func TestExpand_Simple(t *testing.T) {
	m := map[string]string{
		"HOST": "localhost",
		"PORT": "5432",
		"URL":  "postgres://${HOST}:${PORT}/app",
	}
	if err := expandAll(m); err != nil {
		t.Fatal(err)
	}
	want := "postgres://localhost:5432/app"
	if m["URL"] != want {
		t.Errorf("expand: got %q, want %q", m["URL"], want)
	}
}

func TestExpand_DefaultWhenUnset(t *testing.T) {
	m := map[string]string{
		"GREETING": "hello, ${NAME:-world}",
	}
	if err := expandAll(m); err != nil {
		t.Fatal(err)
	}
	if m["GREETING"] != "hello, world" {
		t.Errorf("default expansion: got %q", m["GREETING"])
	}
}

func TestExpand_FailWhenUnset(t *testing.T) {
	m := map[string]string{
		"SECRET": "${MISSING:?env MISSING required}",
	}
	err := expandAll(m)
	if !errors.Is(err, ErrUnresolvedVariable) {
		t.Errorf("expected ErrUnresolvedVariable, got %v", err)
	}
	if !strings.Contains(err.Error(), "MISSING required") {
		t.Errorf("error should include user message, got %v", err)
	}
}

func TestExpand_UnsetBecomesEmpty(t *testing.T) {
	m := map[string]string{"URL": "x=${ABSENT}y"}
	if err := expandAll(m); err != nil {
		t.Fatal(err)
	}
	if m["URL"] != "x=y" {
		t.Errorf("unset variable should become empty, got %q", m["URL"])
	}
}
