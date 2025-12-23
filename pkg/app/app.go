package app

import (
	"context"
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
	BaseConfig *config.BaseConfig
	Location   *time.Location
	Container  *di.Container

	KernelManager *KernelManager

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

	return app.KernelManager.Register(k)
}

// InitKernel выполняет init kernel (один раз)
func (app *App) InitKernel(name string) error {
	if err := app.ensureInit(); err != nil {
		return err
	}

	return app.KernelManager.Init(app, name)
}

// RunKernel запускает kernel
func (app *App) RunKernel(name string) error {
	if err := app.ensureInit(); err != nil {
		return err
	}

	return app.KernelManager.Run(app, name)
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
	app.shutdownTimeout = 30 * time.Second // @TODO можно в конфиг

	if app.errCh == nil {
		app.errCh = make(chan error, 1)
	}

	if app.ctx == nil || app.cancel == nil {
		app.ctx, app.cancel = context.WithCancel(context.Background())
	}

	// Container
	if app.Container == nil {
		app.Container = di.NewContainer()
	}

	// KernelManager
	if app.KernelManager == nil {
		app.KernelManager = NewKernelManager()
	}

	// Config
	{
		if err := config.ReadEnv(); err != nil {
			return err
		}

		baseConfig := &config.BaseConfig{}
		if err := config.InitConfig(baseConfig); err != nil {
			return err
		}

		spew.Dump(baseConfig) // TODO убрать
		app.BaseConfig = baseConfig
	}

	// Timezone
	{
		if app.BaseConfig != nil && app.BaseConfig.TimeZone != "" {
			location, err := time.LoadLocation(app.BaseConfig.TimeZone)
			if err != nil {
				return err
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

// Fail — аварийная остановка
func (app *App) Fail(err error) {
	if err == nil {
		return
	}

	select {
	case app.errCh <- err:
	default:
	}

	if app.cancel != nil {
		app.cancel()
	}
}
