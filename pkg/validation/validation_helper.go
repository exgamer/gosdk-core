package validation

import (
	"encoding/json"
	"errors"
	"github.com/go-playground/validator/v10"
	"github.com/gookit/validate"
	"github.com/iancoleman/strcase"
)

// ValidationErrorsAsMap -возвращает ошибки валидации как map
func ValidationErrorsAsMap(validationErrors validate.Errors) map[string]any {
	eMap := make(map[string]any, len(validationErrors))

	for k, ve := range validationErrors {
		eMap[k] = ve.String()
	}

	return eMap
}

// BindANdValidateStruct - биндит в структуру массив битов и валидирует
func BindANdValidateStruct[T any](byte []byte, i *T) (map[string]string, error) {
	err := json.Unmarshal(byte, i)

	if err != nil {
		return nil, err
	}

	v := validator.New()
	err = v.Struct(i)

	if err != nil {
		var ve validator.ValidationErrors
		out := make(map[string]string, len(ve))

		if errors.As(err, &ve) {
			for _, fe := range ve {
				out[strcase.ToSnake(fe.Field())] = fe.Error()
			}
		}

		return out, nil
	}

	return nil, nil
}
