##  Что доступно из DI после инициализации приложения

Таймзона:

```go
location, err := app.GetLocation(c *di.Container) (*time.Location, error)
```

Базовый конфиг приложения:

```go
baseConfig := app.GetBaseConfig(c *di.Container) (*config.BaseConfig, error)
```