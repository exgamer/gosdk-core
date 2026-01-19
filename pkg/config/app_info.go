package config

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

// AppInfo Данные приложения
type AppInfo struct {
	ServiceName string
	AppEnv      string
	LogLevel    string
}
