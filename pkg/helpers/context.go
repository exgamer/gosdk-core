package helpers

import (
	"context"
	"github.com/exgamer/gosdk-core/pkg/config"
	"github.com/exgamer/gosdk-core/pkg/constants"
)

func GetAppInfoFromContext(ctx context.Context) *config.AppInfo {
	if dbg, ok := ctx.Value(constants.AppInfoKey).(*config.AppInfo); ok {
		return dbg
	}

	return nil
}
