package exception

import (
	"github.com/exgamer/gosdk-core/pkg/constants"
	"github.com/exgamer/gosdk-core/pkg/validation"
	"github.com/go-errors/errors"
	"github.com/gookit/validate"
	"net/http"
)

// AppException Модель данных для описания ошибки
type AppException struct {
	Code          int
	Error         error
	Context       map[string]any
	ServiceCode   int
	TrackInSentry bool
}

func (a *AppException) GetErrorType() string {
	return constants.GetErrorTypeByStatusCode(a.Code)
}

func NewAppException(code int, err error, context map[string]any) *AppException {
	return &AppException{code, err, context, 0, true}
}

func NewInternalServerAppException(err error, context map[string]any) *AppException {
	return &AppException{http.StatusInternalServerError, err, context, 0, true}
}

func NewValidationAppException(context map[string]any) *AppException {
	return &AppException{http.StatusUnprocessableEntity, errors.New("VALIDATION ERROR"), context, 0, true}
}

func NewUntrackableAppException(code int, err error, context map[string]any) *AppException {
	return &AppException{code, err, context, 0, false}
}

func NewValidationAppExceptionFromValidationErrors(validationErrors validate.Errors) *AppException {
	return NewValidationAppException(validation.ValidationErrorsAsMap(validationErrors))
}
