##  Что доступно из DI после инициализации приложения

Таймзона:

```go
location, err := di.GetLocation(c *di.Container) (*time.Location, error)
```

Базовый конфиг приложения:

```go
baseConfig := di.GetBaseConfig(c *di.Container) (*config.BaseConfig, error)
```