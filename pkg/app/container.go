package app

import (
	"github.com/exgamer/gosdk-core/pkg/config"
	"github.com/exgamer/gosdk-core/pkg/di"
	"time"
)

// GetLocation возвращает Location Timezone.
func GetLocation(a *App) (*time.Location, error) {
	c, err := di.Resolve[*time.Location](a.Container)

	if err != nil {
		return nil, err
	}

	return c, nil
}

// GetBaseConfig возвращает BaseConfig.
func GetBaseConfig(a *App) (*config.BaseConfig, error) {
	c, err := di.Resolve[*config.BaseConfig](a.Container)

	if err != nil {
		return nil, err
	}

	return c, nil
}
