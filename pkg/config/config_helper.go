package config

import (
	"encoding/json"
	"errors"
	"fmt"
	structHelper "github.com/exgamer/gosdk-core/pkg/structures"
	"github.com/spf13/viper"
	"os"
	"strings"
)

// LoadEnv читает env файлы по порядку + всегда включает OS ENV.
// Первый найденный файл используется.
func LoadEnv(paths ...string) error {
	viper.Reset()

	viper.AutomaticEnv()
	viper.SetConfigType("env")

	if len(paths) == 0 {
		paths = []string{".env"}
	}

	for _, path := range paths {
		if path == "" {
			continue
		}

		viper.SetConfigFile(path)

		err := viper.ReadInConfig()
		if err == nil {
			return nil
		}

		var notFound viper.ConfigFileNotFoundError
		if errors.As(err, &notFound) || os.IsNotExist(err) {
			continue
		}

		return err
	}

	return nil
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
