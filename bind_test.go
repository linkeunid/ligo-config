package ligo_config

import (
	"errors"
	"testing"
	"time"
)

type dbConfig struct {
	Host     string        `default:"localhost" env:"DB_HOST"            validate:"required"`
	Port     int           `default:"5432"      env:"DB_PORT"            validate:"gt=0"`
	URL      string        `env:"DB_URL"        validate:"omitempty,url"`
	Debug    bool          `default:"false"     env:"DB_DEBUG"`
	Timeout  time.Duration `default:"5s"        env:"DB_TIMEOUT"`
	MaxConns uint32        `default:"10"        env:"DB_MAX"`
	Ratio    float64       `default:"1.5"       env:"DB_RATIO"`
	Tags     []string      `env:"DB_TAGS"`
}

func TestBind_Defaults(t *testing.T) {
	svc := newServiceWith(map[string]string{})
	cfg, err := Bind[dbConfig](svc)
	if err != nil {
		t.Fatalf("Bind: %v", err)
	}
	if cfg.Host != "localhost" {
		t.Errorf("Host = %q, want localhost", cfg.Host)
	}
	if cfg.Port != 5432 {
		t.Errorf("Port = %d, want 5432", cfg.Port)
	}
	if cfg.Timeout != 5*time.Second {
		t.Errorf("Timeout = %v, want 5s", cfg.Timeout)
	}
	if cfg.MaxConns != 10 {
		t.Errorf("MaxConns = %d, want 10", cfg.MaxConns)
	}
	if cfg.Ratio != 1.5 {
		t.Errorf("Ratio = %v, want 1.5", cfg.Ratio)
	}
	if cfg.Debug {
		t.Errorf("Debug = true, want false")
	}
}

func TestBind_FromEnv(t *testing.T) {
	svc := newServiceWith(map[string]string{
		"DB_HOST":    "prod.db",
		"DB_PORT":    "6432",
		"DB_URL":     "postgres://prod.db:6432/app",
		"DB_DEBUG":   "true",
		"DB_TIMEOUT": "30s",
		"DB_MAX":     "100",
		"DB_TAGS":    "primary, hot, fast",
	})
	cfg, err := Bind[dbConfig](svc)
	if err != nil {
		t.Fatalf("Bind: %v", err)
	}

	if cfg.Host != "prod.db" {
		t.Errorf("Host = %q, want prod.db", cfg.Host)
	}
	if cfg.Port != 6432 {
		t.Errorf("Port = %d, want 6432", cfg.Port)
	}
	if !cfg.Debug {
		t.Errorf("Debug should be true")
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", cfg.Timeout)
	}
	if cfg.MaxConns != 100 {
		t.Errorf("MaxConns = %d, want 100", cfg.MaxConns)
	}

	if len(cfg.Tags) != 3 || cfg.Tags[0] != "primary" || cfg.Tags[2] != "fast" {
		t.Errorf("Tags = %v, want [primary hot fast]", cfg.Tags)
	}
}

func TestBind_ValidationFailure(t *testing.T) {
	type strictConfig struct {
		URL string `env:"URL" validate:"required,url"`
	}
	svc := newServiceWith(map[string]string{"URL": "not-a-url"})
	_, err := Bind[strictConfig](svc)
	if !errors.Is(err, ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestBind_InvalidIntWrapsErrInvalidValue(t *testing.T) {
	type c struct {
		N int `env:"N"`
	}
	svc := newServiceWith(map[string]string{"N": "not-a-number"})
	_, err := Bind[c](svc)
	if !errors.Is(err, ErrInvalidValue) {
		t.Errorf("expected ErrInvalidValue, got %v", err)
	}
}

func TestBind_TimeRFC3339(t *testing.T) {
	type c struct {
		When time.Time `env:"WHEN"`
	}
	svc := newServiceWith(map[string]string{"WHEN": "2026-05-17T12:00:00Z"})
	cfg, err := Bind[c](svc)
	if err != nil {
		t.Fatalf("Bind: %v", err)
	}
	want := time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)
	if !cfg.When.Equal(want) {
		t.Errorf("When = %v, want %v", cfg.When, want)
	}
}

func TestMustBind_PanicsOnError(t *testing.T) {
	type strict struct {
		Required string `env:"REQUIRED" validate:"required"`
	}
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustBind should panic on validation failure")
		}
	}()
	svc := newServiceWith(map[string]string{})
	_ = MustBind[strict](svc)
}

func TestBind_NotAStruct(t *testing.T) {
	svc := newServiceWith(nil)
	_, err := Bind[int](svc)
	if err == nil {
		t.Error("Bind with non-struct T should error")
	}
}
