package ligo_config

import "errors"

var (
	// ErrKeyNotFound is returned by MustGet when a key is absent.
	ErrKeyNotFound = errors.New("config: key not found")

	// ErrInvalidValue is returned when a value cannot be coerced to the
	// requested type (e.g. GetInt on a non-numeric string).
	ErrInvalidValue = errors.New("config: invalid value")

	// ErrLoadFailed wraps any failure that occurs while a Loader or
	// .env file is being parsed during Module initialization.
	ErrLoadFailed = errors.New("config: load failed")

	// ErrValidation wraps validator/v10 errors raised after Bind.
	ErrValidation = errors.New("config: validation failed")

	// ErrUnresolvedVariable is returned when ${VAR:?msg} expansion fails
	// because VAR is missing and a fail-on-unset marker is present.
	ErrUnresolvedVariable = errors.New("config: unresolved variable")
)
