package config

import (
	"encoding/json"
	"fmt"
	structHelper "github.com/exgamer/gosdk-core/pkg/structures"
	"github.com/spf13/viper"
	"os"
	"strings"
)

// LoadEnv читает env файлы по порядку + всегда включает OS ENV.
// Первый найденный файл используется.
func LoadEnv(paths ...string) (string, error) {
	viper.Reset()
	viper.SetConfigType("env")

	if len(paths) == 0 {
		paths = []string{
			".env",
		}
	}

	var loadedFrom string

	for _, path := range paths {
		if path == "" {
			continue
		}

		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				continue
			}

			return "", err
		}

		viper.SetConfigFile(path)

		if err := viper.ReadInConfig(); err != nil {
			return "", err
		}

		loadedFrom = path

		break
	}

	// ENV всегда сверху
	viper.AutomaticEnv()

	return loadedFrom, nil
}

// InitConfig Инициализирует конфиг из переменок окружения
func InitConfig[E any](config *E) error {
	err := viper.Unmarshal(config)

	if err != nil {
		fmt.Println(err.Error())
	}

	envKeys := structHelper.GetFieldsAsMapStructureTags(config)
	osEnvMap := make(map[string]any, len(envKeys))

	for _, key := range envKeys {
		if value, exists := os.LookupEnv(key); exists {
			key = strings.ToLower(key)
			osEnvMap[key] = value
		}
	}

	//	// Convert the map to JSON
	jsonData, _ := json.Marshal(osEnvMap)
	// Convert the JSON to a struct
	uErr := json.Unmarshal(jsonData, config)

	if uErr != nil {
		return uErr
	}

	return nil
}
