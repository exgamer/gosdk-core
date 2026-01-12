package exception

func NewAppException(err error, context map[string]any, trackInSentry bool) *AppException {
	return &AppException{err, context, trackInSentry}
}

// AppException Модель данных для описания ошибки
type AppException struct {
	Err           error
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
