package ligo_config

import "context"

// Loader is a programmatic configuration source. Loaders run during Module
// initialization and contribute key/value pairs that override values from
// .env files (but not process env, unless [WithLoadersWin] is set).
type Loader func(ctx context.Context) (map[string]string, error)

// Validator is an optional hook that runs after all sources have been
// merged. Return a non-nil error to abort startup.
type Validator func(*Service) error

type options struct {
	envFiles      []string
	ignoreEnvFile bool
	expand        bool
	loaders       []Loader
	validate      Validator
	loadersWin    bool
}

// Option configures the [Module] returned by [Module] / [ConfigModule].
type Option func(*options)

// WithEnvFiles sets the list of .env files to attempt to load, in order
// from highest precedence to lowest. The first existing file's keys win
// among files. Missing files are skipped silently.
//
// Default: [".env"].
func WithEnvFiles(paths ...string) Option {
	return func(o *options) {
		o.envFiles = append([]string(nil), paths...)
	}
}

// WithIgnoreEnvFile skips .env file loading entirely. Use in containerized
// environments where all configuration comes from the process env or
// explicit loaders.
func WithIgnoreEnvFile(ignore bool) Option {
	return func(o *options) { o.ignoreEnvFile = ignore }
}

// WithExpand enables POSIX-style variable expansion inside values:
//
//   - ${VAR}        — substitute VAR; empty string if unset
//   - ${VAR:-def}   — substitute VAR; "def" if unset or empty
//   - ${VAR:?msg}   — substitute VAR; fail load with msg if unset or empty
//
// Variables resolve against the already-merged map at expansion time, so
// later-loaded sources can reference earlier ones.
//
// Default: false (literal values, no expansion).
func WithExpand(expand bool) Option {
	return func(o *options) { o.expand = expand }
}

// WithLoader appends a programmatic configuration source. Loaders run in
// the order they are registered; later loaders override earlier ones.
//
// By default loaders override .env files but not process env. Use
// [WithLoadersWin] to flip that precedence.
func WithLoader(loader Loader) Option {
	return func(o *options) {
		o.loaders = append(o.loaders, loader)
	}
}

// WithValidate registers a hook that runs after all sources have been
// merged but before the [Service] is exposed to other providers. Returning
// a non-nil error aborts [ligo.App.Run].
func WithValidate(v Validator) Option {
	return func(o *options) { o.validate = v }
}

// WithLoadersWin makes [Loader]-supplied keys override process env. Useful
// in tests where you want to inject a deterministic config map regardless
// of the host's environment.
func WithLoadersWin(win bool) Option {
	return func(o *options) { o.loadersWin = win }
}

func defaultOptions() options {
	return options{
		envFiles: []string{".env"},
		expand:   false,
	}
}
