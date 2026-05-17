package ligo_config

import (
	"context"

	"github.com/linkeunid/ligo"
)

// ModuleName is the registered name of the config module within Ligo's
// module tree. Useful for module composition diagnostics.
const ModuleName = "ligo_config"

// Module returns a Ligo module that loads configuration from .env files,
// programmatic [Loader]s, and process env, then publishes a singleton
// [*Service] into the DI container.
//
// It is the Go equivalent of [ConfigModule.forRoot] from @nestjs/config —
// register it once at the top of your module tree and inject *Service into
// downstream providers.
//
//	app.Register(
//	    ligo_config.Module(
//	        ligo_config.WithEnvFiles(".env.local", ".env"),
//	        ligo_config.WithExpand(true),
//	        ligo_config.WithValidate(func(s *ligo_config.Service) error {
//	            _, err := s.MustGet("DATABASE_URL")
//	            return err
//	        }),
//	    ),
//	    myFeatureModule(),
//	)
//
// Loading happens during OnInit, so failures (missing required keys,
// validator errors, malformed .env files) abort startup before any
// downstream factory is constructed.
//
// [ConfigModule.forRoot]: https://docs.nestjs.com/techniques/configuration#getting-started
func Module(opts ...Option) ligo.Module {
	resolved := defaultOptions()
	for _, opt := range opts {
		opt(&resolved)
	}

	svc := newService()
	bootstrap := &configBootstrap{svc: svc, opts: resolved}

	return ligo.NewModule(
		ModuleName,
		ligo.Providers(
			ligo.Value(svc),
			ligo.HookedSingleton[*configBootstrap](func() *configBootstrap {
				return bootstrap
			}),
		),
	)
}

// Provider returns just the [*Service] provider without the surrounding
// module. Use when you want to embed config setup inside an existing
// module instead of registering a dedicated one.
//
// This variant skips loaders, .env files, and validation — it just
// publishes a [*Service] backed by [os.Environ]. Reach for [Module] when
// you need the full pipeline (env files, expansion, validators).
func Provider() ligo.Provider {
	return ligo.Factory[*Service](func() *Service {
		svc := newService()
		merged, _ := loadSources(context.Background(), options{ignoreEnvFile: true})
		svc.setAll(merged)
		return svc
	})
}

// Load synchronously builds a [*Service] outside the Ligo DI lifecycle.
// Use in main() when you need configuration values BEFORE [ligo.New] —
// for example, to resolve the bind address passed to [ligo.WithAddr],
// which wires at construction time, earlier than Module's OnInit hook.
//
//	svc, err := ligo_config.Load(ligo_config.WithEnvFiles(".env"))
//	if err != nil {
//	    panic(err)
//	}
//	addr := ":" + svc.GetOr("PORT", "8080")
//	app := ligo.New(ligo.WithAddr(addr), ...)
//
// The returned *Service is independent from the one Module produces; they
// read the same sources but do not share state. If you only need config
// at runtime (inside handlers, use cases, providers), prefer injecting
// *Service via [Module] instead.
func Load(opts ...Option) (*Service, error) {
	resolved := defaultOptions()
	for _, opt := range opts {
		opt(&resolved)
	}
	merged, err := loadSources(context.Background(), resolved)
	if err != nil {
		return nil, err
	}
	svc := newService()
	svc.setAll(merged)
	if resolved.validate != nil {
		if err := resolved.validate(svc); err != nil {
			return nil, err
		}
	}
	return svc, nil
}

// MustLoad is the panicking variant of [Load]. Use in main() where a
// missing or malformed config should crash the process before [ligo.New].
func MustLoad(opts ...Option) *Service {
	svc, err := Load(opts...)
	if err != nil {
		panic(err)
	}
	return svc
}

// configBootstrap is a register-only provider whose OnInit hook drives
// the loader pipeline. It exists so config loading happens inside Ligo's
// lifecycle (and errors abort startup) instead of in a factory body.
type configBootstrap struct {
	svc  *Service
	opts options
}

// Register wires OnInit so source loading runs after construction but
// before downstream providers' OnInit hooks.
func (b *configBootstrap) Register(r *ligo.HookRegistry) {
	r.OnInit(b.init)
}

func (b *configBootstrap) init() error {
	merged, err := loadSources(context.Background(), b.opts)
	if err != nil {
		return err
	}
	b.svc.setAll(merged)
	if b.opts.validate != nil {
		if err := b.opts.validate(b.svc); err != nil {
			return err
		}
	}
	return nil
}
