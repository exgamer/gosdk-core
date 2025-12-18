### Пример использования контейнера зависимостей

## Не рекомендуется использовать контейнер вне App
```go

   container := di.NewContainer()

   // Регистрируем сервис
   Register(a.container, config.BaseConfig{ServerAddress: "sdfasdfasdfas"})
   Register(a.container, factories.NewRedisRepositoryFactory(a.redisClient))
   Register(a.container, func() config.BaseConfig {
        return config.BaseConfig{ServerAddress: "sdfasdfasdfas"}
   })

   rf, err := Resolve[*factories.RedisRepositoryFactory](a.container)

   if err != nil {
    spew.Dump(err)
   } else {
    spew.Dump(rf)
   }

   s, err := Resolve[config.BaseConfig](a.container)

   if err != nil {
        spew.Dump(err)
   } else {
       spew.Dump(s)
   }
   
```