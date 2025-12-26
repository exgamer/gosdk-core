package regex

import (
	"errors"
	"regexp"
)

// StringIsPositiveInt - проверяет является ли строка положительным int
func StringIsPositiveInt(param string) error {
	r, err := regexp.Compile(`^[1-9][0-9]{0,17}$`)

	if err != nil {
		return err
	}

	if r.MatchString(param) == false {
		return errors.New("value is not integer")
	}

	return nil
}

// IsLatinCyrillicWithSpaces - проверяет чтоыб строка содержала только кирилицу, латиницу, пробел и цифры
func IsLatinCyrillicWithSpaces(s string) bool {
	re := regexp.MustCompile(`^[A-Za-zА-Яа-яЁё0-9 ]+$`)

	return re.MatchString(s)
}
