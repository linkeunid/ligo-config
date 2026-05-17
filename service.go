// Package ligo_config provides a Ligo-native configuration service inspired
// by [@nestjs/config]. It loads layered configuration from .env files,
// process env, and programmatic loaders, then exposes a [*Service] that
// other Ligo providers consume via dependency injection.
//
//	app.Register(
//	    ligo_config.Module(
//	        ligo_config.WithEnvFiles(".env.local", ".env"),
//	        ligo_config.WithExpand(true),
//	    ),
//	    myFeatureModule(),
//	)
//
//	func NewMyService(cfg *ligo_config.Service) *MyService {
//	    return &MyService{
//	        host: cfg.GetOr("HOST", "localhost"),
//	        port: cfg.MustGetInt("PORT"),
//	    }
//	}
//
// [@nestjs/config]: https://docs.nestjs.com/techniques/configuration
package ligo_config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Service is the merged, immutable view over all configured sources. It
// is registered as a singleton provider by [Module]; inject *Service into
// any factory in the same module tree.
//
// Service is safe for concurrent use.
type Service struct {
	mu     sync.RWMutex
	values map[string]string
}

// newService constructs an empty Service. Call setAll to populate it.
func newService() *Service {
	return &Service{values: map[string]string{}}
}

// setAll replaces the backing map. Used by the loader pipeline; not
// exported because the merged map is final once the Module starts.
func (s *Service) setAll(m map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values = m
}

// Get returns the raw string value for key and a presence flag. Empty
// strings count as present.
func (s *Service) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.values[key]
	return v, ok
}

// GetOr returns the string value for key, or def if the key is absent.
func (s *Service) GetOr(key, def string) string {
	if v, ok := s.Get(key); ok {
		return v
	}
	return def
}

// MustGet returns the string value for key or wraps [ErrKeyNotFound] if
// the key is absent.
func (s *Service) MustGet(key string) (string, error) {
	if v, ok := s.Get(key); ok {
		return v, nil
	}
	return "", fmt.Errorf("%w: %q", ErrKeyNotFound, key)
}

// GetInt parses the value as an int. Returns [ErrKeyNotFound] if absent
// and [ErrInvalidValue] if non-numeric.
func (s *Service) GetInt(key string) (int, error) {
	v, ok := s.Get(key)
	if !ok {
		return 0, fmt.Errorf("%w: %q", ErrKeyNotFound, key)
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return 0, fmt.Errorf("%w: %q=%q: %w", ErrInvalidValue, key, v, err)
	}
	return n, nil
}

// GetIntOr returns the int value for key, or def on miss or parse error.
func (s *Service) GetIntOr(key string, def int) int {
	n, err := s.GetInt(key)
	if err != nil {
		return def
	}
	return n
}

// GetBool parses the value as a bool (1/0/true/false/yes/no/on/off,
// case-insensitive). Returns [ErrKeyNotFound] or [ErrInvalidValue].
func (s *Service) GetBool(key string) (bool, error) {
	v, ok := s.Get(key)
	if !ok {
		return false, fmt.Errorf("%w: %q", ErrKeyNotFound, key)
	}
	b, err := parseBool(v)
	if err != nil {
		return false, fmt.Errorf("%w: %q=%q: %w", ErrInvalidValue, key, v, err)
	}
	return b, nil
}

// GetBoolOr returns the bool value for key, or def on miss or parse error.
func (s *Service) GetBoolOr(key string, def bool) bool {
	b, err := s.GetBool(key)
	if err != nil {
		return def
	}
	return b
}

// GetDuration parses the value via [time.ParseDuration] (e.g. "5s",
// "300ms", "1h30m"). Returns [ErrKeyNotFound] or [ErrInvalidValue].
func (s *Service) GetDuration(key string) (time.Duration, error) {
	v, ok := s.Get(key)
	if !ok {
		return 0, fmt.Errorf("%w: %q", ErrKeyNotFound, key)
	}
	d, err := time.ParseDuration(strings.TrimSpace(v))
	if err != nil {
		return 0, fmt.Errorf("%w: %q=%q: %w", ErrInvalidValue, key, v, err)
	}
	return d, nil
}

// GetDurationOr returns the [time.Duration] value for key, or def on miss
// or parse error.
func (s *Service) GetDurationOr(key string, def time.Duration) time.Duration {
	d, err := s.GetDuration(key)
	if err != nil {
		return def
	}
	return d
}

// All returns a defensive copy of the merged key/value map. Intended for
// debugging and validation hooks; not a hot path.
func (s *Service) All() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string, len(s.values))
	for k, v := range s.values {
		out[k] = v
	}
	return out
}

func parseBool(s string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "t", "true", "y", "yes", "on":
		return true, nil
	case "0", "f", "false", "n", "no", "off", "":
		return false, nil
	}
	return false, errors.New("not a boolean")
}
