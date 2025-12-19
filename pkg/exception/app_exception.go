package exception

// AppException Модель данных для описания ошибки
type AppException struct {
	Error   error
	Context map[string]any
}

func NewAppException(err error, context map[string]any) *AppException {
	return &AppException{err, context}
}
