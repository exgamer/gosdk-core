package exception

import "errors"

type ErrorKind string

const (
	ErrorKindValidation ErrorKind = "validation"
	ErrorKindNotFound   ErrorKind = "not_found"
	ErrorKindForbidden  ErrorKind = "forbidden"
	ErrorKindInternal   ErrorKind = "internal"
)

func NewValidationException(context map[string]any, trackInSentry bool) *AppException {
	return &AppException{
		Err:           errors.New("validation error"),
		Kind:          ErrorKindValidation,
		Context:       context,
		TrackInSentry: trackInSentry,
	}
}

func NewNotFoundException(err error, context map[string]any, trackInSentry bool) *AppException {
	return &AppException{
		Err:           err,
		Kind:          ErrorKindNotFound,
		Context:       context,
		TrackInSentry: trackInSentry,
	}
}

func NewForbiddenException(err error, context map[string]any, trackInSentry bool) *AppException {
	return &AppException{
		Err:           err,
		Kind:          ErrorKindForbidden,
		Context:       context,
		TrackInSentry: trackInSentry,
	}
}

func NewAppException(err error, context map[string]any, trackInSentry bool) *AppException {
	return &AppException{err, ErrorKindInternal, context, trackInSentry}
}

// AppException Модель данных для описания ошибки
type AppException struct {
	Err           error
	Kind          ErrorKind
	Context       map[string]any
	TrackInSentry bool
}

func (e *AppException) Error() string {
	if e.Err == nil {
		return "unknown application exception"
	}

	return e.Err.Error()
}

func (e *AppException) Unwrap() error {
	return e.Err
}
