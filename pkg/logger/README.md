# logger

Лёгкий потокобезопасный логгер-обёртка над стандартным `log.Logger` из
Go с: - уровнями логирования (TRACE/DEBUG/INFO/WARN/ERROR/FATAL) -
форматированием в одну строку: `timestamp LEVEL message` - возможностью
подменить `*log.Logger` - управлением уровнем через `LOG_LEVEL` 

## Быстрый старт

``` go
logger.Init()
logger.SetLevel(context, logger.LevelInfo)

logger.Info(context, "service started")
logger.Debug(context, "debug message")
logger.Error(context, "error message")
logger.Warning(context, "warn message")
logger.Fatal(context, "fatal message")
logger.Trace(context, "trace message")
```

## Уровни логирования

Поддерживаемые уровни: `trace`, `debug`, `info`, `warn`, `error`,
`fatal`

Лог выводится, если уровень сообщения \>= текущего уровня логгера.

## ENV

``` bash
LOG_LEVEL=info
```

``` go
logger.SetLevel(logger.ParseLevel(os.Getenv("LOG_LEVEL")))
```

## Подмена логгера

``` go
custom := log.New(os.Stdout, "[service] ", 0)
logger.SetLogger(custom)
```

## Рекомендации

-   prod: info / warn
-   staging: debug
-   local: debug / trace
