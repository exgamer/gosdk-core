package exception

// AppException Модель данных для описания ошибки
type AppException struct {
	Error         error
	Context       map[string]any
	TrackInSentry bool
}

func NewAppException(err error, context map[string]any, trackInSentry bool) *AppException {
	return &AppException{err, context, trackInSentry}
}
