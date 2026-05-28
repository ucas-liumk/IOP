package localization

import (
	"context"
	"strings"
)

// Bundle is the i18n storage contract.
type Bundle interface {
	Lookup(locale, key string) (string, bool)
}

// MapBundle is an in-memory Bundle for tests.
func MapBundle(data map[string]map[string]string) Bundle { return mapBundle{data: data} }

type mapBundle struct{ data map[string]map[string]string }

func (m mapBundle) Lookup(locale, key string) (string, bool) {
	if loc := m.data[locale]; loc != nil {
		v, ok := loc[key]
		return v, ok
	}
	return "", false
}

// Service exposes T(ctx, key, args...). args is [k, v, k, v, ...].
type Service struct {
	bundle        Bundle
	defaultLocale string
}

func NewService(b Bundle, defaultLocale string) *Service {
	if defaultLocale == "" {
		defaultLocale = "zh-CN"
	}
	return &Service{bundle: b, defaultLocale: defaultLocale}
}

// T returns the translation for key, or key itself if missing.
// Args are [name, value, name, value...] pairs replacing {name} placeholders.
func (s *Service) T(ctx context.Context, key string, args ...string) string {
	locale := s.defaultLocale
	if v, ok := localeFromContext(ctx); ok {
		locale = v
	}
	tpl, ok := s.bundle.Lookup(locale, key)
	if !ok {
		return key
	}
	out := tpl
	for i := 0; i+1 < len(args); i += 2 {
		out = strings.ReplaceAll(out, "{"+args[i]+"}", args[i+1])
	}
	return out
}

// localeFromContext is populated by middleware in M2 (Accept-Language parsing).
// In M1 it always returns "".
type localeCtxKey struct{}

func WithLocale(ctx context.Context, locale string) context.Context {
	return context.WithValue(ctx, localeCtxKey{}, locale)
}

func localeFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(localeCtxKey{}).(string)
	return v, ok
}
