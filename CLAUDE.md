# CLAUDE.md

## Behavioral Foundation

1. Don't assume. Don't hide confusion. Surface tradeoffs.
2. Minimum code that solves the problem. Nothing speculative.
3. Touch only what you must. Clean up only your own mess.
4. Define success criteria. Loop until verified.

## Project

A [Ligo](https://github.com/linkeunid/ligo) application scaffolded by
[ligo-cli](https://github.com/linkeunid/ligo-cli). Ligo is a modular Go
framework with lightweight DI inspired by NestJS.

- Framework: `github.com/linkeunid/ligo`
- Generator: `ligo new` / `ligo generate`
- License: MIT

## Commands

```bash
go build ./...                          # Build
go run ./cmd/...                        # Run
go test ./...                           # Run tests
go test -v ./...                        # Verbose tests
go test -race ./...                     # Race detector
go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out
go test -bench=. -benchmem ./...        # Benchmarks
golangci-lint run                       # Lint (config: .golangci.yml)
gofumpt -w .                            # Format (stricter than gofmt)
govulncheck ./...                       # CVE scan
go mod tidy                             # Tidy deps
```

## Go Best Practices (do)

- **Small, focused packages.** A package's name should describe a single
  responsibility. If you can't name it cleanly, it's doing too much.
- **Accept interfaces, return structs.** Consumers depend on the smallest
  surface possible; producers expose concrete types so behavior can grow.
- **Errors are values.** Wrap with `fmt.Errorf("doing X: %w", err)`. Check
  with `errors.Is` / `errors.As`, never `==` or type assertion.
- **Context first.** `func F(ctx context.Context, ...)` — always the first
  parameter, never stored in a struct.
- **Use `any`, not `interface{}`.** Modern Go.
- **Pre-allocate slices** when the size is known: `make([]T, 0, n)`.
- **`fmt.Errorf("%w", err)` to wrap, `errors.New` for sentinel errors.**

## Go Best Practices (don't)

- **Don't use `init()` for application logic.** It's untestable. Use a
  factory function and register it in a module's providers instead.
- **Don't `panic` in library code.** Return an error. Reserve panics for
  truly unrecoverable programmer mistakes (e.g. invariant violation).
- **Don't ignore errors.** `_ = doX()` only when you've actually decided
  the error doesn't matter, and leave a comment explaining why.
- **Don't share mutable state across goroutines without synchronization.**
  Channels for ownership transfer; `sync.Mutex` for shared mutation;
  `sync.RWMutex` only when reads vastly outnumber writes.
- **Don't put business logic in HTTP handlers / controllers.** They
  translate request ↔ response and delegate. Use cases own the rules.
- **Don't reach for global variables.** Inject via Ligo's DI container.

## Ligo Practices (do)

- **One module per bounded context.** `user`, `auth`, `billing` — each its
  own `ligo.NewModule(...)` with providers + controllers + middleware.
- **Constructor injection.** Factories take their dependencies as
  parameters; Ligo's DI resolves them.
  ```go
  func NewUserService(repo *UserRepository, log ligo.Logger) *UserService
  ligo.Factory[*UserService](NewUserService)
  ```
- **Compile-time-safe hooks** via `HookedFactory[T]` / `HookedController` —
  the `Register(r *ligo.HookRegistry)` method takes method values, so
  typos become compile errors.
- **`HookedSingleton[T]` for register-only providers.** Schedulers, broker
  handler registrations, background workers — anything no other provider
  depends on. A plain `HookedFactory` would never instantiate and its
  hooks would silently no-op.
- **Validate at the edge.** Use `ligo.ValidationPipe(&Dto{})` on the route
  and `ligo.ValidatedBody[Dto](ctx)` in the handler. The use case can
  then trust its input.
- **Resolve dependencies with `ligo.MustResolve[T](app)` only after
  `app.Run()`.** Prefer `ligo.Resolve[T]` (returns `(T, error)`) when
  the failure is recoverable; reserve `MustResolve` for cases where a
  missing provider really should crash the process.
- **Pagination in the framework.** `ctx.Paginate(20, 100)` and
  `ctx.Paginated(items, page, perPage, total)` — don't roll your own.
- **Query binding in the framework.** `ctx.BindQuery(&filter)` with
  `query:"name"` tags. Don't parse `ctx.Request().URL.Query()` by hand.

## Ligo Practices (don't)

- **Don't store DI-resolved singletons in package-level vars.** Resolve at
  construction time and pass through the constructor chain.
- **Don't depend on resolution order between providers in the same
  module.** OnInit hooks run after construction; do cross-provider setup
  there, not in factories.
  As of ligo v0.10.0 these hooks run sequentially in registration
  order — opt back into the legacy parallel execution with
  `ligo.WithParallelHooks()` only if you have many independent I/O-bound
  providers and ordering does not matter.
- **Don't put `Handle[T,R](...)` / `On[T](...)` calls in a factory body.**
  They mutate broker state and need the broker to be connected — do them
  in `OnBootstrap` via `HookedSingleton`.
- **Don't bypass middleware by composing handlers manually.** Use
  `ligo.NewChainRouter(r.Group(prefix))` so middleware order is explicit.
- **Don't pin a Ligo minor version forever.** Track minor releases —
  they add API surface without breaking changes.

## Static Analysis

Every Ligo project ships `.golangci.yml` (schema v2) enabling
`errcheck`, `govet` (with `shadow` and `nilness`), `staticcheck`,
`unused`, `gofumpt`, `misspell`, `unconvert`, `unparam`, `revive`,
`bodyclose`, `errorlint`, `nolintlint`, `whitespace`, `tagalign`, `gci`.

`gci` enforces uniform import order: stdlib, third-party, local —
separated by blank lines, in that order. The `prefix(...)` matches this
project's module path so your own packages always group as "local".

`infertypeargs` (the gopls analyzer that flags generic calls like
`microservices.Handle[Foo, *Bar](...)` where the handler signature
already determines the type args) is **not** wired into `golangci-lint
v2` — the analyzer lives in gopls's internal package and v2 cannot
import it. The check still runs in your editor via gopls, so strip the
redundant type args when the warning lights up. See
https://pkg.go.dev/golang.org/x/tools/gopls/internal/analysis/infertypeargs.

Install the toolchain once:

```bash
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
go install honnef.co/go/tools/cmd/staticcheck@latest
go install mvdan.cc/gofumpt@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
go install github.com/daixiang0/gci@latest
go install github.com/4meepo/tagalign/cmd/tagalign@latest
go install golang.org/x/tools/gopls@latest
```

`tagalign` keeps struct tags aligned with two-space separators. The
columns must line up across all fields of a multi-field struct:

```go
type CreateUserInput struct {
    Name  string `json:"name"  validate:"required,min=2,max=100"`
    Email string `json:"email" validate:"required,email"`
}
```

Auto-fix: `tagalign -fix -sort $(find . -name "*.go" -not -path "./vendor/*")`.

Pre-merge checklist:

```bash
gci write --skip-generated -s standard -s default --custom-order .
gofumpt -w .                                                     # formatting
tagalign -fix -sort $(find . -name '*.go' -not -path './vendor/*') # tag columns
go test -race ./...     # tests + race detector
golangci-lint run       # static checks (incl. gci, gofumpt, tagalign)
govulncheck ./...       # CVE scan
```

## CI

`.github/workflows/ci.yml` runs `golangci-lint` (v2), `go test -race`,
and `govulncheck` on every push to `main` and every pull request,
pinned to Node-24 action versions (checkout v6, setup-go v6,
golangci-lint-action v9). Lint catches gci / gofumpt / errcheck /
errorlint drift before merge.
