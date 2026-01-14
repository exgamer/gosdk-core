package di

import (
	"github.com/exgamer/gosdk-core/pkg/config"
	"time"
)

// GetLocation возвращает Location Timezone.
func GetLocation(c *Container) (*time.Location, error) {
	e, err := Resolve[*time.Location](c)

	if err != nil {
		return nil, err
	}

	return e, nil
}

// GetBaseConfig возвращает BaseConfig.
func GetBaseConfig(c *Container) (*config.BaseConfig, error) {
	e, err := Resolve[*config.BaseConfig](c)

	if err != nil {
		return nil, err
	}

	return e, nil
}
