package config

import (
	"encoding/json"
	"fmt"
	structHelper "github.com/exgamer/gosdk-core/pkg/structures"
	"github.com/spf13/viper"
	"os"
	"strings"
)

// ReadEnv Чтение переменок окружения
func ReadEnv() error {
	root, err := os.Getwd()

	if err != nil {
		return err
	}

	viper.AddConfigPath(root)
	viper.SetConfigType("env")
	viper.SetConfigName(".env")
	viper.AutomaticEnv()
	err = viper.ReadInConfig()

	if err != nil {
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
	uErr := json.Unmarshal(jsonData, &config)

	if uErr != nil {
		return uErr
	}

	return nil
}
