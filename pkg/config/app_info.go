package config

import (
	"context"
	"github.com/exgamer/gosdk-core/pkg/constants"
)

func GetInstanceAppInfo(appConfig *BaseConfig) *AppInfo {
	appInfo := &AppInfo{}
	appInfo.AppEnv = "UNKNOWN"
	appInfo.ServiceName = "UNKNOWN"

	if appConfig != nil {
		appInfo.AppEnv = appConfig.AppEnv
		appInfo.ServiceName = appConfig.Name
		appInfo.LogLevel = appConfig.LogLevel
	}

	return appInfo
}

func GetAppInfoFromContext(ctx context.Context) *AppInfo {
	if v := ctx.Value(constants.AppInfoKey); v != nil {
		if ai, ok := v.(*AppInfo); ok {
			return ai
		}
	}

	return nil
}

// AppInfo Данные приложения
type AppInfo struct {
	ServiceName string
	AppEnv      string
	LogLevel    string
}
