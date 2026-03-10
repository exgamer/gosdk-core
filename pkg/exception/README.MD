# exception: AppException

`AppException` — это обёртка над обычной ошибкой (`error`), которая добавляет:
- `Context` — дополнительные данные (для логов/отладки/ответов)
- `TrackInSentry` — флаг, отправлять ли ошибку в Sentry
- поддержку `errors.As / errors.Is` через `Unwrap()`

---

## Структура

```go
type AppException struct {
    Err           error
	Kind          ErrorKind
    Context       map[string]any
    TrackInSentry bool
}
```

`AppException` реализует интерфейс `error`:

- `Error()` — текст ошибки
- `Unwrap()` — возвращает первичную ошибку (`Err`)

---

## Создание

```go
ex := exception.NewAppException(err, map[string]any{
    "user_id": userID,
    "order_id": orderID,
}, true)
```

- `TrackInSentry=true` — можно отправлять в Sentry
- `TrackInSentry=false` — не отправлять (ожидаемые/бизнес ошибки)

---

## Использование как обычного error

`AppException` можно возвращать и пробрасывать как `error`:

```go
func (s *Service) Do() error {
    if err := s.repo.Save(); err != nil {
        return exception.NewAppException(err, map[string]any{
            "layer": "repo",
        }, true)
    }
    return nil
}
```

---

## Доступ к AppException через errors.As

```go
import "errors"

var appEx *exception.AppException
if errors.As(err, &appEx) {
    // appEx.Context
    // appEx.TrackInSentry
}
```

---

## errors.Is и Unwrap

Так как реализован `Unwrap()`, можно проверять первопричину:

```go
if errors.Is(err, sql.ErrNoRows) {
    // первичная ошибка была sql.ErrNoRows
}
```

---

## Рекомендации по Context

- храните только полезные поля для диагностики (id, параметры, stage)
- не кладите секреты: токены, пароли, cookies, персональные данные

Пример:

```go
ctx := map[string]any{
    "request_id": rid,
    "city_id":    cityID,
    "method":     "GetById",
}
```

---

## Пример использования 

### Логирование
```go
var appEx *exception.AppException
if errors.As(err, &appEx) {
    logger.Error("app error", appEx.Context)
} else {
    logger.Error(err.Error())
}
```

### Sentry
```go
var appEx *exception.AppException
if errors.As(err, &appEx) && !appEx.TrackInSentry {
    return
}
sentry.CaptureException(err)
```

---

# Error Handling

В проекте используется двухуровневая модель ошибок:

- **AppException** — ошибка уровня приложения (domain/application)
- **HttpException** — ошибка уровня HTTP transport

Это позволяет **разделить бизнес-логику и транспорт**, чтобы сервисы не зависели от HTTP.

---

# 1. AppException (application/domain layer)

`AppException` используется в сервисах и доменной логике.

```go
    NewValidationException
    NewNotFoundException
    NewForbiddenException
    NewAppException
```


## Зачем нужен AppException

- прокидывать контекст по слоям (service/repo/api) без кастомных структур
- централизованно решать: логировать/отправлять в Sentry или нет
- не терять первопричину ошибки благодаря `Unwrap()`
