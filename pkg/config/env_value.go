package config

import (
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var (
	durationType         = reflect.TypeFor[time.Duration]()
	jsonUnmarshalerType  = reflect.TypeFor[json.Unmarshaler]()
	textUnmarshalerType  = reflect.TypeFor[encoding.TextUnmarshaler]()
	errUnsupportedFloats = errors.New("NaN and Inf are not supported")
)

// envValue Приводит значение переменной окружения key к типу поля, в которое json положит этот ключ.
// Приводятся int*, uint*, float*, bool и указатели на них, правила разбора — как у viper для .env;
// остальные поля получают строку, как раньше.
func envValue(configType reflect.Type, key, value string) (any, error) {
	fieldType, ok := scalarFieldType(configType, strings.ToLower(key))
	if !ok {
		return value, nil
	}

	typed, err := parseScalar(fieldType, value)
	if err != nil {
		return nil, fmt.Errorf("config: env %s: invalid %s value: %w", key, fieldType.Kind(), err)
	}

	return typed, nil
}

// scalarFieldType Тип поля для ключа jsonKey, если значение надо привести из строки.
// Duration, типы со своим UnmarshalJSON/UnmarshalText и поля с опцией json ",string" не приводятся.
func scalarFieldType(configType reflect.Type, jsonKey string) (reflect.Type, bool) {
	field, quoted, ok := jsonField(configType, jsonKey)
	if !ok || quoted {
		return nil, false
	}

	fieldType := field.Type
	if fieldType.Kind() == reflect.Pointer {
		fieldType = fieldType.Elem()
	}

	if fieldType == durationType || hasOwnUnmarshaler(fieldType) {
		return nil, false
	}

	return fieldType, isScalarKind(fieldType.Kind())
}

// jsonField Поле верхнего уровня, в которое encoding/json декодирует ключ jsonKey:
// точное совпадение имени, иначе первое без учёта регистра. quoted — у тега есть опция ",string".
func jsonField(configType reflect.Type, jsonKey string) (field reflect.StructField, quoted bool, ok bool) {
	var folded *reflect.StructField

	foldedQuoted := false

	for i := range configType.NumField() {
		candidate := configType.Field(i)
		if !candidate.IsExported() || candidate.Anonymous {
			continue
		}

		name, opts, _ := strings.Cut(candidate.Tag.Get("json"), ",")
		if name == "-" && opts == "" {
			continue
		}

		if name == "" {
			name = candidate.Name
		}

		if name == jsonKey {
			return candidate, hasStringOption(opts), true
		}

		if folded == nil && strings.EqualFold(name, jsonKey) {
			folded = &candidate
			foldedQuoted = hasStringOption(opts)
		}
	}

	if folded == nil {
		return reflect.StructField{}, false, false
	}

	return *folded, foldedQuoted, true
}

func hasStringOption(opts string) bool {
	for opt := range strings.SplitSeq(opts, ",") {
		if opt == "string" {
			return true
		}
	}

	return false
}

func hasOwnUnmarshaler(t reflect.Type) bool {
	ptr := reflect.PointerTo(t)

	return ptr.Implements(jsonUnmarshalerType) || ptr.Implements(textUnmarshalerType)
}

func isScalarKind(kind reflect.Kind) bool {
	switch kind {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

// parseScalar Разбирает строку как mapstructure с WeaklyTypedInput (так viper читает .env):
// пустая строка — нулевое значение, целые — с префиксами 0x/0o/0b.
func parseScalar(t reflect.Type, value string) (any, error) {
	if value == "" && t.Kind() == reflect.Bool {
		return false, nil
	}

	if value == "" {
		value = "0"
	}

	switch t.Kind() {
	case reflect.Bool:
		return strconv.ParseBool(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.ParseInt(value, 0, t.Bits())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.ParseUint(value, 0, t.Bits())
	default:
		return parseFloat(value, t.Bits())
	}
}

// parseFloat NaN и Inf не передать через JSON, поэтому это ошибка значения.
func parseFloat(value string, bitSize int) (any, error) {
	f, err := strconv.ParseFloat(value, bitSize)
	if err != nil {
		return nil, err
	}

	if math.IsNaN(f) || math.IsInf(f, 0) {
		return nil, errUnsupportedFloats
	}

	return f, nil
}
