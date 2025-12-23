package app

import (
	"context"
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/exgamer/gosdk-core/pkg/config"
	"github.com/exgamer/gosdk-core/pkg/di"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func NewApp() *App {
	return &App{}
}

type App struct {
	BaseConfig      *config.BaseConfig
	Location        *time.Location
	Container       *di.Container
	kernels         map[string]KernelInterface
	initedKernels   map[string]struct{}
	startedKernels  map[string]struct{}
	stopHooks       []func(ctx context.Context) error
	shutdownTimeout time.Duration
	once            sync.Once
	shutdownOnce    sync.Once
	mu              sync.Mutex
	initErr         error

	ctx    context.Context
	cancel context.CancelFunc
	errCh  chan error
}

func (app *App) GetContext() context.Context {
	return app.ctx
}

// RegisterKernel регистрирует kernel
func (app *App) RegisterKernel(k KernelInterface) error {
	if err := app.ensureInit(); err != nil {
		return err
	}

	name := k.Name()
	if name == "" {
		return errors.New("kernel name is empty")
	}

	app.mu.Lock()
	defer app.mu.Unlock()

	if _, exists := app.kernels[name]; exists {
		return errors.New("kernel already registered: " + name)
	}

	app.kernels[name] = k

	return nil
}

// InitKernel выполняет init kernel (один раз)
func (app *App) InitKernel(name string) error {
	if err := app.ensureInit(); err != nil {
		return err
	}

	// 1) быстрые проверки под локом
	app.mu.Lock()
	k, ok := app.kernels[name]
	if !ok || k == nil {
		app.mu.Unlock()
		return fmt.Errorf("kernel not registered: %s", name)
	}

	if _, inited := app.initedKernels[name]; inited {
		app.mu.Unlock()

		return nil // уже инициализирован — ок
	}

	// резервируем "как будто inited", чтобы параллельный InitKernel не вошёл
	app.initedKernels[name] = struct{}{}
	app.mu.Unlock()

	rollback := func() {
		app.mu.Lock()
		delete(app.initedKernels, name)
		app.mu.Unlock()
	}

	// 2) тяжёлая часть без лока
	if err := k.Init(app); err != nil {
		rollback()
		return fmt.Errorf("init %s: %w", name, err)
	}

	return nil
}

// RunKernel запускает модуль
func (app *App) RunKernel(name string) error {
	if err := app.ensureInit(); err != nil {
		return err
	}

	// гарантируем init
	if err := app.InitKernel(name); err != nil {
		return err
	}

	// 1) резервируем старт под локом
	app.mu.Lock()

	kernel, ok := app.kernels[name]
	if !ok || kernel == nil {
		app.mu.Unlock()

		return fmt.Errorf("kernel not registered: %s", name)
	}

	if _, exists := app.startedKernels[name]; exists {
		app.mu.Unlock()

		return fmt.Errorf("kernel already started: %s", name)
	}

	app.startedKernels[name] = struct{}{}
	app.mu.Unlock()

	rollbackStarted := func() {
		app.mu.Lock()
		delete(app.startedKernels, name)
		app.mu.Unlock()
	}

	// 2) старт без лока
	if err := kernel.Start(app); err != nil {
		ctx, cancel := context.WithTimeout(app.ctx, app.shutdownTimeout)
		defer cancel()
		_ = kernel.Stop(ctx)

		rollbackStarted()
		return fmt.Errorf("start %s: %w", name, err)
	}

	app.AddStopHook(func(ctx context.Context) error {
		return kernel.Stop(ctx)
	})

	return nil
}

// ensureInit гарантирует initApp 1 раз и возвращает ошибку инициализации.
func (app *App) ensureInit() error {
	app.once.Do(func() {
		app.initErr = app.initApp()
	})

	return app.initErr
}

// initApp инициализация приложения
func (app *App) initApp() error {
	app.stopHooks = make([]func(ctx context.Context) error, 0)
	app.shutdownTimeout = 30 * time.Second //@TODO возможно вынести в конфиг

	if app.errCh == nil {
		app.errCh = make(chan error, 1) // буфер 1, чтобы не блокировать
	}

	if app.ctx == nil || app.cancel == nil {
		app.ctx, app.cancel = context.WithCancel(context.Background())
	}

	// Инициализация контейнера
	if app.Container == nil {
		app.Container = di.NewContainer()
	}

	if app.kernels == nil {
		app.kernels = make(map[string]KernelInterface)
	}

	if app.initedKernels == nil {
		app.initedKernels = make(map[string]struct{})
	}

	if app.startedKernels == nil {
		app.startedKernels = make(map[string]struct{})
	}

	// Инициализация конфига апп
	{
		envErr := config.ReadEnv()

		if envErr != nil {
			return envErr
		}

		baseConfig := &config.BaseConfig{}
		err := config.InitConfig(baseConfig)

		if err != nil {
			return err
		}

		spew.Dump(baseConfig) //TODO удалить

		app.BaseConfig = baseConfig
	}

	// Инициализиация тайм зоны
	{
		if app.BaseConfig.TimeZone != "" {
			location, lErr := time.LoadLocation(app.BaseConfig.TimeZone)

			if lErr != nil {
				return lErr
			}

			app.Location = location
		}
	}

	return nil
}

// AddStopHook Добавить функцию, которая будет вызвана на shutdown
func (app *App) AddStopHook(hook func(ctx context.Context) error) {
	app.mu.Lock()
	app.stopHooks = append(app.stopHooks, hook)
	app.mu.Unlock()
}

// WaitForShutdown graceful
func (app *App) WaitForShutdown() {
	app.shutdownOnce.Do(func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(stop)

		select {
		case <-stop:
			log.Println("Shutting down application (signal)...")
		case err := <-app.errCh:
			log.Printf("Shutting down application (fatal error): %v", err)
		case <-app.ctx.Done():
			log.Println("Shutting down application (context canceled)...")
		}

		if app.cancel != nil {
			app.cancel()
		}

		// второй сигнал — форс
		go func() {
			<-stop
			log.Println("Forced shutdown.")
			os.Exit(1)
		}()

		ctx, cancel := context.WithTimeout(context.Background(), app.shutdownTimeout)
		defer cancel()

		app.mu.Lock()
		hooks := append([]func(context.Context) error(nil), app.stopHooks...)
		app.mu.Unlock()

		for i := len(hooks) - 1; i >= 0; i-- {
			if err := hooks[i](ctx); err != nil {
				log.Printf("Shutdown hook error: %v", err)
			}
		}

		log.Println("Application stopped gracefully.")
	})
}

func (app *App) Fail(err error) {
	if err == nil {
		return
	}
	// положим ошибку один раз
	select {
	case app.errCh <- err:
	default:
	}
	if app.cancel != nil {
		app.cancel()
	}
}
