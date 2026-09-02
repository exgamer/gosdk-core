# errorreporter

Абстракция над системой трекинга ошибок (Sentry, Rollbar и т.п.), не
привязанная к конкретному вендору и к HTTP-транспорту. `gosdk-core` не
тянет зависимость на sentry-go или любой другой SDK - только интерфейс
`Reporter` и глобальный регистр, по аналогии с `pkg/logger`.

Конкретную реализацию регистрирует адаптер (например `SentryKernel` из
`gosdk-sentry-core`) во время старта приложения. Пока адаптер не
зарегистрирован, `errorreporter` работает как no-op - вызовы безопасны
из любого кода, включая тесты.

## Зачем

Иногда ошибку не нужно пробрасывать наверх и превращать в 500-й ответ -
у вызывающего кода есть штатный fallback. Но потерять её молча тоже
нельзя - нужно знать, что fallback вообще срабатывает. Пример: Redis
недоступен -> идём в БД напрямую, но факт падения кэша должен долететь
до Sentry.

## Быстрый старт

``` go
val, err := redisHelper.GetStruct(key)
if err != nil {
    errorreporter.CaptureSoft(ctx, err, map[string]string{
        "component": "redis_cache",
        "fallback":  "db",
    })

    return repo.getFromDB(ctx, id) // ошибку наверх не пробрасываем
}
```

## API

``` go
// произвольный уровень и доп. данные
errorreporter.Capture(ctx, err, errorreporter.Options{
    Level: errorreporter.LevelError,
    Tags:  map[string]string{"component": "rabbit_consumer"},
    Extra: map[string]any{"message_id": msgID},
})

// некритичная ошибка с fallback-путём - вернувшуюся ошибку можно игнорировать
errorreporter.CaptureSoft(ctx, err, map[string]string{"component": "redis_cache"})

// ошибку нужно и отрепортить, и пробросить наверх - одной строкой
if err != nil {
    return errorreporter.CaptureError(ctx, err, map[string]string{"component": "rabbit_consumer"})
}
```

## Единый механизм и защита от дублей

Все три функции (`Capture`/`CaptureSoft`/`CaptureError`) возвращают ту же
ошибку, но уже помеченную как отправленная. Если такая ошибка попадёт в
`Capture` повторно на более высоком уровне (например Domain отрепортил и
пробросил, а Transport при формировании HTTP-ответа тоже пытается
отрепортить), повторной отправки в трекер не произойдёт - маркер
проверяется через `errorreporter.WasReported(err)`.

Это то, что делает пакет единой точкой репортинга для всего SDK: не
только "проглотить и отрепортить" (Redis -> fallback в БД), но и "выбить
ошибку и отрепортить" (например паника/ошибка в rabbit-консьюмере, где
нет HTTP-транспорта, который отрепортит её за вас). Любой адаптер верхнего
уровня, который тоже умеет репортить в Sentry (HTTP middleware, consumer),
должен звать этот же `errorreporter.Capture`, а не дёргать `sentry-go`
напрямую - иначе дедуп не сработает и появится параллельный, ничем не
связанный источник событий в Sentry.

Если ошибка уже отправлена, повторный `Capture`/`CaptureSoft`/`CaptureError`
возвращает тот же `err` без изменений (без лишней вложенности обёртки) и
ничего никуда не шлёт. Если нужно осознанно отправить её ещё раз (например
эскалация: сначала `CaptureSoft` на первой неудачной попытке, затем
`CaptureError` после исчерпания ретраев) - передайте `Options.Force: true`.

## Регистрация реализации (для авторов адаптеров)

``` go
type sentryReporter struct{}

func (sentryReporter) Capture(ctx context.Context, err error, opts errorreporter.Options) {
    // ... sentry.CaptureException(err) с обогащением из ctx
}

func init() {
    errorreporter.SetReporter(&sentryReporter{})
}
```

На практике регистрация происходит не в `init()`, а в `Kernel.Init(a *app.App)`
конкретного адаптера - см. `gosdk-sentry-core`.

## Важно

- Это инструмент для **некритичных** ошибок с явным fallback. Ошибки,
  которые должны прервать выполнение и стать HTTP-ответом, по-прежнему
  просто возвращаются (`error`) и обрабатываются транспортным слоем как
  раньше.
- Пакеты уровня Infrastructure (`gosdk-redis-core` и т.п.) сами `CaptureSoft`
  не вызывают - они остаются "тупыми" примитивами и просто возвращают
  `error`. Решение "проглотить ошибку и сделать fallback" - ответственность
  вызывающего кода приложения (репозитория), а не SDK-хелпера.
