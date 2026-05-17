package ligo_config

import (
	"errors"
	"testing"
	"time"
)

func newServiceWith(m map[string]string) *Service {
	s := newService()
	s.setAll(m)
	return s
}

func TestService_Get(t *testing.T) {
	s := newServiceWith(map[string]string{"FOO": "bar", "EMPTY": ""})

	v, ok := s.Get("FOO")
	if !ok || v != "bar" {
		t.Errorf("Get(FOO) = (%q,%v), want (bar,true)", v, ok)
	}
	v, ok = s.Get("EMPTY")
	if !ok || v != "" {
		t.Errorf("Get(EMPTY) should report present empty, got (%q,%v)", v, ok)
	}
	_, ok = s.Get("MISSING")
	if ok {
		t.Error("Get(MISSING) should report absent")
	}
}

func TestService_GetOr(t *testing.T) {
	s := newServiceWith(map[string]string{"FOO": "bar"})
	if got := s.GetOr("FOO", "x"); got != "bar" {
		t.Errorf("GetOr(FOO,x) = %q, want bar", got)
	}
	if got := s.GetOr("MISSING", "default"); got != "default" {
		t.Errorf("GetOr(MISSING,default) = %q, want default", got)
	}
}

func TestService_MustGet(t *testing.T) {
	s := newServiceWith(map[string]string{"FOO": "bar"})
	v, err := s.MustGet("FOO")
	if err != nil || v != "bar" {
		t.Errorf("MustGet(FOO) = (%q,%v), want (bar,nil)", v, err)
	}
	_, err = s.MustGet("MISSING")
	if !errors.Is(err, ErrKeyNotFound) {
		t.Errorf("MustGet(MISSING) should wrap ErrKeyNotFound, got %v", err)
	}
}

func TestService_GetInt(t *testing.T) {
	s := newServiceWith(map[string]string{"N": "42", "BAD": "x"})
	if n, err := s.GetInt("N"); err != nil || n != 42 {
		t.Errorf("GetInt(N) = (%d,%v), want (42,nil)", n, err)
	}
	if _, err := s.GetInt("BAD"); !errors.Is(err, ErrInvalidValue) {
		t.Errorf("GetInt(BAD) should wrap ErrInvalidValue, got %v", err)
	}
	if _, err := s.GetInt("MISSING"); !errors.Is(err, ErrKeyNotFound) {
		t.Errorf("GetInt(MISSING) should wrap ErrKeyNotFound, got %v", err)
	}
	if got := s.GetIntOr("MISSING", 7); got != 7 {
		t.Errorf("GetIntOr(MISSING,7) = %d, want 7", got)
	}
	if got := s.GetIntOr("BAD", 7); got != 7 {
		t.Errorf("GetIntOr(BAD,7) = %d, want 7", got)
	}
}

func TestService_GetBool(t *testing.T) {
	cases := map[string]bool{
		"true": true, "TRUE": true, "1": true, "yes": true, "on": true, "T": true,
		"false": false, "FALSE": false, "0": false, "no": false, "off": false,
	}
	for in, want := range cases {
		s := newServiceWith(map[string]string{"B": in})
		if got, err := s.GetBool("B"); err != nil || got != want {
			t.Errorf("GetBool(%q) = (%v,%v), want (%v,nil)", in, got, err, want)
		}
	}
	s := newServiceWith(map[string]string{"BAD": "maybe"})
	if _, err := s.GetBool("BAD"); !errors.Is(err, ErrInvalidValue) {
		t.Errorf("GetBool(BAD) should wrap ErrInvalidValue, got %v", err)
	}
	if got := s.GetBoolOr("MISSING", true); !got {
		t.Errorf("GetBoolOr(MISSING,true) = %v, want true", got)
	}
}

func TestService_GetDuration(t *testing.T) {
	s := newServiceWith(map[string]string{"D": "5s", "BAD": "soon"})
	d, err := s.GetDuration("D")
	if err != nil || d != 5*time.Second {
		t.Errorf("GetDuration(D) = (%v,%v), want (5s,nil)", d, err)
	}
	if _, err := s.GetDuration("BAD"); !errors.Is(err, ErrInvalidValue) {
		t.Errorf("GetDuration(BAD) should wrap ErrInvalidValue, got %v", err)
	}
	if got := s.GetDurationOr("MISSING", time.Minute); got != time.Minute {
		t.Errorf("GetDurationOr(MISSING,1m) = %v, want 1m", got)
	}
}

func TestService_AllIsCopy(t *testing.T) {
	s := newServiceWith(map[string]string{"FOO": "bar"})
	snapshot := s.All()
	snapshot["FOO"] = "mutated"
	if v, _ := s.Get("FOO"); v != "bar" {
		t.Errorf("All() should return a defensive copy; service mutated to %q", v)
	}
}
