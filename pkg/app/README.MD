# Application Core — Usage Guide

Данный документ описывает, как использовать `App` и модульную архитектуру  
(инициализация, регистрация модулей, запуск, graceful shutdown).

---

## Общая идея архитектуры

Приложение построено вокруг структуры `App`, которая:

- управляет жизненным циклом модулей
- предоставляет общий `context`
- обрабатывает graceful shutdown (`SIGINT` / `SIGTERM`)
- поддерживает фатальные ошибки из модулей
- гарантирует корректный порядок остановки модулей (**LIFO**)

Каждый модуль реализует интерфейс `ModuleInterface`.

---

## Интерфейс модуля

```go
type KernelInterface interface {
    Name() string
    Register(app *App) error
    Start(app *App) error
    Stop(ctx context.Context) error
}
```

---

## Жизненный цикл приложения

1. Создание `App`
2. Регистрация модулей (`RegisterKernel`)
3. Запуск модулей (`RunKernel`)
4. Ожидание сигнала завершения (`WaitForShutdown`)
5. Остановка модулей в обратном порядке

---

## Пример main.go

```go
func main() {
    appInstance, err := app.NewApp()
    if err != nil {
        log.Fatal(err)
    }

    if err := appInstance.RegisterKernel(&modules.CommonKernel{}); err != nil {
        log.Fatal(err)
    }

    if err := appInstance.RegisterKernel(&modules.HttpKernel{}); err != nil {
        log.Fatal(err)
    }

    if err := appInstance.RunKernel("common"); err != nil {
        log.Fatal(err)
    }

    if err := appInstance.RunKernel("http"); err != nil {
        log.Fatal(err)
    }

    appInstance.WaitForShutdown()
}
```

---

## RegisterModule

- Вызывает `kernel.Register(app)`
- Гарантирует, что модуль регистрируется только один раз
- Потокобезопасен

---

## RunModule

- Вызывает `kernel.Start(app)`
- Регистрирует `StopHook`
- Не блокирует поток

---

## Graceful shutdown

- Обрабатывает `SIGINT / SIGTERM`
- Вызывает `Stop(ctx)` для каждого модуля
- Использует timeout (по умолчанию 30 секунд)

---

## Context приложения

- Отменяется при shutdown
- Отменяется при `app.Fail(err)`
- Используется для фоновых goroutine

---

## Рекомендации

**Register():**
- загрузка конфигурации
- регистрация зависимостей
- инициализация клиентов

**Start():**
- запуск HTTP / gRPC
- запуск background workers

**Stop():**
- Shutdown(ctx)
- Close()
- Flush()

---

## Итог

Архитектура обеспечивает предсказуемый lifecycle, корректный graceful shutdown  
и готовность к Docker / Kubernetes.
