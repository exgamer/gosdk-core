package exception

type ErrorKind string

const (
	ErrorKindValidation ErrorKind = "validation"
	ErrorKindNotFound   ErrorKind = "not_found"
	ErrorKindForbidden  ErrorKind = "forbidden"
	ErrorKindInternal   ErrorKind = "internal"
)

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
