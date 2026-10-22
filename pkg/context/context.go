package context

import (
	"context"

	"github.com/exgamer/gosdk-core/pkg/config"
)

// appInfoKey Ключ AppInfo в контексте; свой тип исключает коллизию со строковыми ключами.
type appInfoKey struct{}

// WithAppInfo Кладёт AppInfo в контекст; читать — GetAppInfoFromContext.
func WithAppInfo(ctx context.Context, info *config.AppInfo) context.Context {
	return context.WithValue(ctx, appInfoKey{}, info)
}

// GetAppInfoFromContext Читает AppInfo, положенный WithAppInfo; nil — если ctx == nil,
// значения нет или оно не *config.AppInfo. Строковый ключ "app_info" не читается.
func GetAppInfoFromContext(ctx context.Context) *config.AppInfo {
	if ctx == nil {
		return nil
	}

	if info, ok := ctx.Value(appInfoKey{}).(*config.AppInfo); ok {
		return info
	}

	return nil
}
