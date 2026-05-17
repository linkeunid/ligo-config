# ligo-config

Layered configuration for [Ligo](https://github.com/linkeunid/ligo), inspired
by [@nestjs/config](https://docs.nestjs.com/techniques/configuration).
Loads `.env` files, process environment, and programmatic loaders into a
single injectable `*Service`, with typed binding into structs and
go-playground/validator support.

[![Go Version](https://img.shields.io/badge/go-1.25+-blue)](https://go.dev/dl)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Tests](https://img.shields.io/badge/tests-31%20passing-brightgreen)](https://github.com/linkeunid/ligo-config)
[![Coverage](https://img.shields.io/badge/coverage-82.0%25-brightgreen)](https://github.com/linkeunid/ligo-config)

## Install

```bash
go get github.com/linkeunid/ligo-config
```

## Quick start

```go
package main

import (
    "github.com/linkeunid/ligo"
    "github.com/linkeunid/ligo/adapters/echo"
    ligo_config "github.com/linkeunid/ligo-config"
)

func main() {
    app := ligo.New(
        ligo.WithRouter(echo.NewAdapter()),
        ligo.WithAddr(":8080"),
    )

    app.Register(
        ligo_config.Module(
            ligo_config.WithEnvFiles(".env.local", ".env"),
            ligo_config.WithExpand(true),
        ),
        myModule(),
    )

    _ = app.Run()
}

func myModule() ligo.Module {
    return ligo.NewModule("my",
        ligo.Providers(
            ligo.Factory[*MyService](NewMyService),
        ),
    )
}

func NewMyService(cfg *ligo_config.Service) *MyService {
    return &MyService{
        host: cfg.GetOr("HOST", "localhost"),
        port: cfg.GetIntOr("PORT", 8080),
    }
}
```

### Eager loading (`Load` / `MustLoad`)

When you need configuration values BEFORE `ligo.New` — e.g. to resolve
the bind address for `ligo.WithAddr`, which wires at construction time,
earlier than `Module`'s `OnInit` hook — use `Load`:

```go
svc, err := ligo_config.Load(ligo_config.WithEnvFiles(".env"))
if err != nil {
    panic(err)
}
addr := ":" + svc.GetOr("PORT", "8080")

app := ligo.New(ligo.WithAddr(addr), /* … */)
app.Register(ligo_config.Module(ligo_config.WithEnvFiles(".env")), /* … */)
```

`MustLoad` is the panicking variant for the same scenario.

## Source precedence

Sources merge during `OnInit` in this order, lowest precedence first:

1. `.env` files (listed via `WithEnvFiles`; first file wins among files)
2. Programmatic `Loader`s (registered via `WithLoader`)
3. Process environment (`os.Environ`)

Flip step 2 above step 3 with `WithLoadersWin(true)` — useful in tests
where you want a deterministic config map regardless of the host's env.

Missing `.env` files are not errors. This lets `.env.local` override
`.env` without forcing both to exist.

## API

### Reading values

```go
cfg.Get("KEY")                       // (string, bool) — empty counts as present
cfg.GetOr("KEY", "default")          // string
cfg.MustGet("KEY")                   // (string, error) — ErrKeyNotFound if absent
cfg.GetInt("KEY")                    // (int, error)
cfg.GetIntOr("KEY", 42)              // int
cfg.GetBool("KEY")                   // (bool, error) — 1/0/true/false/yes/no/on/off
cfg.GetBoolOr("KEY", false)          // bool
cfg.GetDuration("KEY")               // (time.Duration, error) — "5s", "300ms", "1h30m"
cfg.GetDurationOr("KEY", time.Second)
cfg.All()                            // map[string]string — defensive copy
```

### Typed binding

`Bind[T]` decodes the merged config into a fresh `*T` using struct tags
and validates the result with go-playground/validator.

```go
type DBConfig struct {
    Host    string        `default:"localhost" env:"DB_HOST"    validate:"required"`
    Port    int           `default:"5432"      env:"DB_PORT"    validate:"gt=0"`
    URL     string        `env:"DB_URL"        validate:"omitempty,url"`
    Timeout time.Duration `default:"5s"        env:"DB_TIMEOUT"`
    Tags    []string      `env:"DB_TAGS"`
}

cfg, err := ligo_config.Bind[DBConfig](svc)
```

Supported field kinds: `string`, `bool`, all `int*` and `uint*` widths,
`float32/64`, `time.Duration`, `time.Time` (RFC3339), and slices of any
of those (split by `,`).

### Namespaced injection (`registerAs`)

`Namespace[T]` registers `*T` as a DI provider, calling `Bind` under the
hood. Downstream factories receive the typed config directly:

```go
func DatabaseModule() ligo.Module {
    return ligo.NewModule("database",
        ligo.Providers(
            ligo_config.Namespace[DBConfig](),
            ligo.Factory[*Repo](NewRepo),
        ),
    )
}

func NewRepo(cfg *DBConfig) *Repo { /* … */ }
```

For custom binding logic (cross-field derivation, runtime defaults), use
`NamespaceFn`:

```go
ligo_config.NamespaceFn[DBConfig](func(svc *ligo_config.Service) (*DBConfig, error) {
    return &DBConfig{
        URL: fmt.Sprintf("postgres://%s:%d/%s",
            svc.GetOr("DB_HOST", "localhost"),
            svc.GetIntOr("DB_PORT", 5432),
            svc.GetOr("DB_NAME", "app")),
    }, nil
})
```

### Variable expansion

`WithExpand(true)` enables POSIX-style interpolation in values:

```env
HOST=localhost
PORT=5432
DATABASE_URL=postgres://${HOST}:${PORT}/app
GREETING=hello, ${NAME:-world}
JWT_SECRET=${JWT_SECRET:?env JWT_SECRET required}
```

- `${VAR}` — substitute `VAR`; empty if unset
- `${VAR:-default}` — substitute `VAR`; `default` if unset or empty
- `${VAR:?message}` — substitute `VAR`; fail startup with `message` if unset

References resolve against the already-merged map, so later sources can
expand values defined by earlier ones.

### Custom loaders

`WithLoader(fn)` adds a programmatic source. Use it to pull config from
Consul, Vault, S3, JSON files, anywhere not covered by `.env`:

```go
ligo_config.WithLoader(func(ctx context.Context) (map[string]string, error) {
    return fetchFromVault(ctx, "secret/app")
})
```

Multiple loaders run in registration order; later loaders override
earlier ones. By default, loaders override `.env` files but not process
env — flip that with `WithLoadersWin(true)`.

### Validation hook

`WithValidate(fn)` runs after sources merge but before downstream
providers initialize. Return a non-nil error to abort startup:

```go
ligo_config.WithValidate(func(s *ligo_config.Service) error {
    if _, err := s.MustGet("DATABASE_URL"); err != nil {
        return err
    }
    if _, err := s.MustGet("JWT_SECRET"); err != nil {
        return err
    }
    return nil
})
```

## See also

- [Ligo](https://github.com/linkeunid/ligo) — the framework this plugs into
- [the-basic sample](https://github.com/linkeunid/sample/tree/main/the-basic) — minimal Ligo app you can clone and run

## License

MIT — see [LICENSE](LICENSE).
