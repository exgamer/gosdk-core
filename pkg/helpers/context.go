package helpers

import (
	"context"
	"gitlab.almanit.kz/jmart/gosdk/pkg/config"
	"gitlab.almanit.kz/jmart/gosdk/pkg/constants"
)

func GetAppInfoFromContext(ctx context.Context) *config.AppInfo {
	if dbg, ok := ctx.Value(constants.AppInfoKey).(*config.AppInfo); ok {
		return dbg
	}

	return nil
}
