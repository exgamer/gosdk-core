## ❗ Работа с ошибками

Базовая структура ошибки:

```go
type AppException struct {
    Error   error
    Context map[string]any
}
```

Пример:

```go
return exception.NewAppException(
    errors.New("user not found"),
    map[string]any{"user_id": id},
)
```