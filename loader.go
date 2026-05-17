package ligo_config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/joho/godotenv"
)

// loadSources merges all configured sources into a single map in this
// precedence order (later overrides earlier when loadersWin is false,
// which matches @nestjs/config behavior):
//
//  1. .env files (first file in the list wins among files)
//  2. programmatic Loaders, in registration order
//  3. process env (os.Environ)
//
// With loadersWin=true, the order becomes 1, 3, 2 so loaders can override
// process env (useful in tests).
func loadSources(ctx context.Context, opts options) (map[string]string, error) {
	merged := map[string]string{}

	// Layer 1: .env files, lowest precedence, first-file-wins among files.
	if !opts.ignoreEnvFile {
		for i := len(opts.envFiles) - 1; i >= 0; i-- {
			path := opts.envFiles[i]
			fileMap, err := readEnvFile(path)
			if err != nil {
				return nil, fmt.Errorf("%w: %s: %w", ErrLoadFailed, path, err)
			}
			for k, v := range fileMap {
				merged[k] = v
			}
		}
	}

	applyLoaders := func() error {
		for i, ldr := range opts.loaders {
			if ldr == nil {
				continue
			}
			m, err := ldr(ctx)
			if err != nil {
				return fmt.Errorf("%w: loader #%d: %w", ErrLoadFailed, i, err)
			}
			for k, v := range m {
				merged[k] = v
			}
		}
		return nil
	}

	applyProcessEnv := func() {
		for _, kv := range os.Environ() {
			eq := strings.IndexByte(kv, '=')
			if eq <= 0 {
				continue
			}
			merged[kv[:eq]] = kv[eq+1:]
		}
	}

	if opts.loadersWin {
		applyProcessEnv()
		if err := applyLoaders(); err != nil {
			return nil, err
		}
	} else {
		if err := applyLoaders(); err != nil {
			return nil, err
		}
		applyProcessEnv()
	}

	if opts.expand {
		if err := expandAll(merged); err != nil {
			return nil, err
		}
	}

	return merged, nil
}

// readEnvFile returns the parsed contents of an env file, or an empty map
// if the file does not exist (missing files are not errors — they let
// .env.local override .env without forcing both to exist).
func readEnvFile(path string) (map[string]string, error) {
	m, err := godotenv.Read(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	return m, nil
}

// expandRe matches ${VAR}, ${VAR:-default}, and ${VAR:?error message}.
var expandRe = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(?::([-?])([^}]*))?\}`)

// expandAll rewrites every value in m in place, substituting ${VAR}
// references with values from m itself. References to unset variables
// become empty strings unless the :?msg form is used.
func expandAll(m map[string]string) error {
	for k, v := range m {
		expanded, err := expandOne(v, m)
		if err != nil {
			return fmt.Errorf("expanding %q: %w", k, err)
		}
		m[k] = expanded
	}
	return nil
}

func expandOne(value string, m map[string]string) (string, error) {
	var firstErr error
	out := expandRe.ReplaceAllStringFunc(value, func(match string) string {
		groups := expandRe.FindStringSubmatch(match)
		name := groups[1]
		op := groups[2]
		extra := groups[3]

		v, ok := m[name]
		if ok && v != "" {
			return v
		}
		switch op {
		case "-":
			return extra
		case "?":
			if firstErr == nil {
				msg := extra
				if msg == "" {
					msg = "variable " + name + " is unset"
				}
				firstErr = fmt.Errorf("%w: %s", ErrUnresolvedVariable, msg)
			}
			return ""
		default:
			return ""
		}
	})
	if firstErr != nil {
		return "", firstErr
	}
	return out, nil
}
