package validation

import (
	"errors"
	"net/mail"
	"regexp"
	"strings"
)

// CheckValidPhone - проверка номера телефона
func CheckValidPhone(phone string) bool {
	re := regexp.MustCompile(`^\+7 \(\d{3}\)-\d{3}-\d{2}-\d{2}$`)

	return re.MatchString(phone)
}

// нормализация номера телефона
// "+7 (775)-557-70-41"  ->  77755577041,
// "+77755577041"        ->  77755577041,
// "87755577041"         ->  77755577041,
// "8(775)557-70-41"     ->  77755577041,
// ""                    ->  The phone number does not match the format
func NormalizePhoneNumber(phone string) (string, error) {
	re := regexp.MustCompile(`\D*(?:\+?7|8)?\D*(\d{1,3})\D*(\d{3})\D*(\d{3})\D*(\d{2})\D*(\d{2})`)
	match := re.FindStringSubmatch(phone)

	if len(match) != 0 {
		digits := strings.Join(match[1:], "")
		// Если номер начинается с "8", заменяем его на "7"
		if digits[0] == '8' {
			digits = "7" + digits[1:]
		}

		return digits, nil
	}

	return "", errors.New("The phone number does not match the format")
}

// CheckValidEmail - проверка адреса email
func CheckValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)

	return err == nil
}
