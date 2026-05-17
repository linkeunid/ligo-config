package ligo_config

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

// validatorOnce holds a shared validator instance. Lazy-initialized so
// importing ligo_config does not eagerly construct a validator (matching
// other parts of the Ligo ecosystem).
var validatorInstance *validator.Validate

// Bind decodes the configuration into a fresh *T by reading struct field
// tags. Supported tags:
//
//   - env:"KEY"        — source key (required for the field to be set)
//   - default:"value"  — used if the env key is unset
//   - validate:"..."   — go-playground/validator rules, applied after decode
//
// Supported field types: string, bool, int/int8/int16/int32/int64, uint
// variants, float32, float64, [time.Duration], [time.Time] (RFC3339), and
// slices of any of those (split by `,`).
//
//	type DBConfig struct {
//	    Host string `env:"DB_HOST" default:"localhost" validate:"required"`
//	    Port int    `env:"DB_PORT" default:"5432"`
//	    URL  string `env:"DB_URL"  validate:"required,url"`
//	}
//
//	cfg, err := ligo_config.Bind[DBConfig](svc)
func Bind[T any](svc *Service) (*T, error) {
	var zero T
	rv := reflect.ValueOf(&zero).Elem()
	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("ligo_config.Bind: T must be a struct, got %s", rv.Kind())
	}
	if err := bindStruct(rv, svc); err != nil {
		return nil, err
	}
	if err := runValidator(&zero); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrValidation, err)
	}
	return &zero, nil
}

// MustBind is the panicking variant of [Bind]. Use in package-level
// initialization where a missing config is genuinely fatal.
func MustBind[T any](svc *Service) *T {
	v, err := Bind[T](svc)
	if err != nil {
		panic(err)
	}
	return v
}

func bindStruct(rv reflect.Value, svc *Service) error {
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		if !field.IsExported() {
			continue
		}
		fv := rv.Field(i)

		// Nested struct (not time.Time) — recurse without prefix translation.
		if fv.Kind() == reflect.Struct && field.Type != reflect.TypeOf(time.Time{}) {
			if err := bindStruct(fv, svc); err != nil {
				return err
			}
			continue
		}

		key := field.Tag.Get("env")
		if key == "" {
			continue
		}

		raw, ok := svc.Get(key)
		if !ok || raw == "" {
			if def, hasDef := field.Tag.Lookup("default"); hasDef {
				raw = def
			} else {
				continue
			}
		}

		if err := setField(fv, raw); err != nil {
			return fmt.Errorf("field %s (env %q): %w", field.Name, key, err)
		}
	}
	return nil
}

func setField(fv reflect.Value, raw string) error {
	if !fv.CanSet() {
		return fmt.Errorf("unsettable field")
	}

	switch fv.Kind() {
	case reflect.String:
		fv.SetString(raw)

	case reflect.Bool:
		b, err := parseBool(raw)
		if err != nil {
			return fmt.Errorf("%w: %q is not a bool", ErrInvalidValue, raw)
		}
		fv.SetBool(b)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Treat time.Duration specially (it's an int64 under the hood).
		if fv.Type() == reflect.TypeOf(time.Duration(0)) {
			d, err := time.ParseDuration(raw)
			if err != nil {
				return fmt.Errorf("%w: %q is not a duration", ErrInvalidValue, raw)
			}
			fv.SetInt(int64(d))
			return nil
		}
		n, err := strconv.ParseInt(raw, 10, fv.Type().Bits())
		if err != nil {
			return fmt.Errorf("%w: %q is not an int", ErrInvalidValue, raw)
		}
		fv.SetInt(n)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(raw, 10, fv.Type().Bits())
		if err != nil {
			return fmt.Errorf("%w: %q is not a uint", ErrInvalidValue, raw)
		}
		fv.SetUint(n)

	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(raw, fv.Type().Bits())
		if err != nil {
			return fmt.Errorf("%w: %q is not a float", ErrInvalidValue, raw)
		}
		fv.SetFloat(f)

	case reflect.Slice:
		return setSlice(fv, raw)

	case reflect.Struct:
		if fv.Type() == reflect.TypeOf(time.Time{}) {
			t, err := time.Parse(time.RFC3339, raw)
			if err != nil {
				return fmt.Errorf("%w: %q is not RFC3339", ErrInvalidValue, raw)
			}
			fv.Set(reflect.ValueOf(t))
			return nil
		}
		return fmt.Errorf("unsupported struct field type: %s", fv.Type())

	default:
		return fmt.Errorf("unsupported field kind: %s", fv.Kind())
	}
	return nil
}

func setSlice(fv reflect.Value, raw string) error {
	parts := strings.Split(raw, ",")
	out := reflect.MakeSlice(fv.Type(), len(parts), len(parts))
	for i, p := range parts {
		p = strings.TrimSpace(p)
		if err := setField(out.Index(i), p); err != nil {
			return fmt.Errorf("slice[%d]: %w", i, err)
		}
	}
	fv.Set(out)
	return nil
}

func runValidator(target any) error {
	if validatorInstance == nil {
		validatorInstance = validator.New(validator.WithRequiredStructEnabled())
	}
	return validatorInstance.Struct(target)
}
