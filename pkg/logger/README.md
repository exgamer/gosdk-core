# Global Logger (stdlib)

Простой **глобальный логгер** на стандартном пакете `log`, чтобы вызывать логи **из любого места кода** без DI.

## Возможности

- Singleton (глобальный экземпляр)
- Потокобезопасная инициализация
- Уровни логирования: `DEBUG | INFO | ERROR`
- Подмена логгера в `main` (файл, buffer, `io.Discard`)
- Подходит для сервисов, CLI, SDK

---

## Быстрый старт

### Инициализация

```go
func main() {
    app.Info("service started")
    app.Debug("debug message")
    app.Error("something went wrong")
}
```

### Управление уровнем логов

По умолчанию уровень `INFO`.

```bash
LOG_LEVEL=DEBUG ./app
LOG_LEVEL=ERROR ./app
```

---

## Подмена логгера

### Логирование в файл

```go
f, err := os.OpenFile("app.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
if err != nil {
    panic(err)
}
defer f.Close()

app.SetLogger(log.New(f, "", 0))
```

### Отключить логирование (no-op)

```go
app.SetLogger(log.New(io.Discard, "", 0))
```

### Использование в тестах

```go
buf := &bytes.Buffer{}
app.SetLogger(log.New(buf, "", 0))

app.Info("hello")

t.Log(buf.String())
```

---

## API

### Init(serviceName string)

Инициализация глобального логгера.  
Вызывается **один раз** при старте приложения.

### SetLogger(logger *log.Logger)

Подменяет глобальный логгер.

### Логирование

```go
app.Debug("debug message")
app.Info("info message")
app.Error("error message")
```

---

## Формат логов

```
2026-01-08 13:02:11.421 INFO service=catalog-service-go service started
```

---

## Лицензия

MIT
