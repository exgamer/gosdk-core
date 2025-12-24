##  Что доступно из DI после инициализации приложения

Таймзона:

```go
location, err := app.GetLocation(a *App) (*time.Location, error)
```

Базовый конфиг приложения:

```go
baseConfig := app.GetBaseConfig(a *App) (*config.BaseConfig, error)
```