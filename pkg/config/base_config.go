package config

// BaseConfig Основной конфиг приложения
type BaseConfig struct {
	Name          string `mapstructure:"APP_NAME" json:"app_name"`
	ContainerName string `mapstructure:"CONTAINER_NAME" json:"container_name"`
	AppEnv        string `mapstructure:"APP_ENV"    json:"app_env"`
	Version       string `mapstructure:"APP_VERSION" json:"app_version"`
	TimeZone      string `mapstructure:"TIMEZONE"    json:"timezone"`
	Debug         bool   `mapstructure:"DEBUG"    json:"debug"`
}
