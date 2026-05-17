package ligo_config

import (
	"github.com/linkeunid/ligo"
)

// NamespaceFactory is a custom builder for a typed configuration block.
// It receives the merged [*Service] and returns a populated *T.
type NamespaceFactory[T any] func(svc *Service) (*T, error)

// Namespace returns a [ligo.Provider] that produces a *T by calling
// [Bind] on the injected [*Service]. Other factories in the same module
// can then receive the typed config directly:
//
//	type DBConfig struct {
//	    Host string `env:"DB_HOST" default:"localhost"`
//	    Port int    `env:"DB_PORT" default:"5432"`
//	}
//
//	func DatabaseModule() ligo.Module {
//	    return ligo.NewModule("database",
//	        ligo.Providers(
//	            ligo_config.Namespace[DBConfig](),
//	            ligo.Factory[*Repo](NewRepo),
//	        ),
//	    )
//	}
//
//	func NewRepo(cfg *DBConfig) *Repo { /* … */ }
//
// This mirrors [registerAs] from @nestjs/config — types act as the
// namespace label so each typed block is independently injectable.
//
// [registerAs]: https://docs.nestjs.com/techniques/configuration#using-the-configservice
func Namespace[T any]() ligo.Provider {
	return ligo.Factory[*T](func(svc *Service) (*T, error) {
		return Bind[T](svc)
	})
}

// NamespaceFn is the same as [Namespace] but lets you supply a custom
// factory instead of going through [Bind]. Use when the env tag model
// doesn't fit (cross-field derivation, runtime feature flags, etc.).
//
//	ligo_config.NamespaceFn[DBConfig](func(svc *ligo_config.Service) (*DBConfig, error) {
//	    return &DBConfig{
//	        URL: fmt.Sprintf("postgres://%s:%d/%s",
//	            svc.GetOr("DB_HOST", "localhost"),
//	            svc.GetIntOr("DB_PORT", 5432),
//	            svc.GetOr("DB_NAME", "app")),
//	    }, nil
//	})
func NamespaceFn[T any](fn NamespaceFactory[T]) ligo.Provider {
	return ligo.Factory[*T](func(svc *Service) (*T, error) {
		return fn(svc)
	})
}
