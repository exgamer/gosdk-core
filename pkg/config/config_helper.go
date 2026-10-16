package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/spf13/viper"

	structHelper "github.com/exgamer/gosdk-core/pkg/structures"
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

// InitConfig Инициализирует конфиг из переменок окружения.
// Значение из окружения процесса приводится к типу поля (int*, uint*, float*, bool);
// неверное значение — ошибка с именем переменной, остальные поля при этом заполняются.
func InitConfig[E any](config *E) error {
	err := viper.Unmarshal(config)

	if err != nil {
		fmt.Println(err.Error())
	}

	envKeys := structHelper.GetFieldsAsMapStructureTags(config)
	configType := reflect.TypeOf(config).Elem()
	osEnvMap := make(map[string]any, len(envKeys))

	var errs []error

	for _, key := range envKeys {
		value, exists := os.LookupEnv(key)
		if !exists {
			continue
		}

		typed, vErr := envValue(configType, key, value)
		if vErr != nil {
			errs = append(errs, vErr)

			continue
		}

		osEnvMap[strings.ToLower(key)] = typed
	}

	jsonData, mErr := json.Marshal(osEnvMap)
	if mErr != nil {
		return errors.Join(append(errs, fmt.Errorf("config: marshal env: %w", mErr))...)
	}

	uErr := json.Unmarshal(jsonData, config)
	if len(errs) == 0 {
		return uErr
	}

	return errors.Join(append(errs, uErr)...)
}
