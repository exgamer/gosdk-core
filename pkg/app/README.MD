# Application Core — Usage Guide

Данный документ описывает, как использовать `App` и модульную архитектуру  
(инициализация, регистрация модулей, запуск, graceful shutdown).

---

## 🧠 Концепция

### App

`App` — это корень приложения.  
Он отвечает за:
- инициализацию конфигурации
- создание DI контейнера
- регистрацию и запуск ядер
- регистрацию и запуск бизнес модулей
- корректное завершение работы

---

## 🧱 Ядра приложения (Kernel)

Ядро — это логический модуль приложения для расширения программы (HTTP, Брокеры сообщений, работа с БД и тп)

## Интерфейс ядра

```go
type KernelInterface interface {
    Name() string
    Register(app *App) error
    Start(app *App) error
    Stop(ctx context.Context) error
}
```

---

## 🧱 Модули приложения

Модуль — это логический бизнес-блок:
- регистрирует свои зависимости
- подключает роуты / обработчики
- не управляет lifecycle сервиса напрямую

### Интерфейс модуля

```go
type ModuleInterface interface {
    Name() string
	Init(app *App) error
}
```

> ❗ `Name()` **должен быть статическим** и уникальным

---

### Пример модуля

```go
type CommonModule struct{}

func (CommonModule) Name() string {
    return "common"
}

func (m CommonModule) Register(app *app.App) error {
    di.Register(app.Container, &ClockService{})
	
    return nil
}
```

---

## Жизненный цикл приложения

1. Создание `App`
2. Регистрация и инициализация ядер (`RegisterAndInitKernels`)
3. Запуск ядер (`RunAll`)
4. Регистрация и инициализация бизнес модулей (`RegisterAndInitModules`)
6. Ожидание сигнала завершения (`WaitForShutdown`)

---

## Пример main.go

```go
func main() {
    appInstance, err := app.NewApp()
    if err != nil {
        log.Fatal(err)
    }

    err := appInstance.RegisterAndInitKernels(
        &http.HttpKernel{},
    )
    if err != nil {
        panic(err)
    }
    
    err = appInstance.RegisterAndInitModules(
        &CommonModule{},
    )
    
    if err != nil {
        panic(err)
    }
    
    if err := a.All(); err != nil {
        panic(err)
    }
    
    appInstance.WaitForShutdown()
}
```

---

## 🧩 Dependency Injection (pkg/di)

Контейнер поддерживает:
- singleton зависимости
- фабрики (lazy)
- автоматическое разрешение зависимостей по типу

[Примеры работы с DI](../di/README.MD)

## Итог

Архитектура обеспечивает предсказуемый lifecycle, корректный graceful shutdown  
и готовность к Docker / Kubernetes.
