package app

import (
	"context"
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/exgamer/gosdk-core/pkg/config"
	"github.com/exgamer/gosdk-core/pkg/di"
	ginHelper "github.com/exgamer/gosdk-core/pkg/gin"
	"github.com/exgamer/gosdk-core/pkg/metricapp"
	"github.com/exgamer/gosdk-core/pkg/tracer"
	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var once sync.Once

func NewApp() *App {
	return &App{}
}

type App struct {
	TraceClient           *tracer.Tracer
	BaseConfig            *config.BaseConfig
	Router                *gin.Engine
	Location              *time.Location
	Metrics               *metricapp.Metrics
	Container             *di.Container
	PrepareConfigsFunc    func(app *App) error
	PrepareComponentsFunc func(app *App) error
	PrepareHttpFunc       func(app *App) error
	PrepareConsumerFunc   func(app *App) error
	stopHooks             []func(ctx context.Context) error
	shutdownTimeout       time.Duration
}

func (app *App) InitBaseConfig() (*config.BaseConfig, error) {
	baseConfig := &config.BaseConfig{HandlerTimeout: 30}
	err := config.InitConfig(baseConfig)

	if err != nil {
		return nil, err
	}

	spew.Dump(baseConfig)

	return baseConfig, nil
}

func (app *App) initConfig() error {
	envErr := config.ReadEnv()

	if envErr != nil {
		fmt.Println(envErr.Error())
	}

	baseConfig, err := app.InitBaseConfig()

	if err != nil {
		return err
	}

	app.BaseConfig = baseConfig

	if app.PrepareConfigsFunc != nil {
		if err := app.PrepareConfigsFunc(app); err != nil {
			return err
		}
	}

	return nil
}

// InitTraceClient - инициализация трейсера
func (app *App) initTraceClient() error {
	traceClient, err := tracer.InitTraceClient()

	if err != nil {
		fmt.Println("Соединение с трассировкой - ошибка : ", err.Error())
	}

	app.TraceClient = traceClient

	return nil
}

// initMetrics инициализация http метрик
func (app *App) initHttpMetrics() {
	app.Metrics = metricapp.InitMetrics(app.BaseConfig.Name)
}

// initApp инициализация приложения
func (app *App) initApp() {
	app.stopHooks = make([]func(ctx context.Context) error, 0)
	app.shutdownTimeout = 10 * time.Second

	if app.Container == nil {
		app.Container = di.NewContainer()
	}

	// Инициализация конфига апп
	{
		err := app.initConfig()

		if err != nil {
			log.Fatalf(err.Error())
		}
	}

	// Инициализиация тайм зоны
	{
		if app.BaseConfig.TimeZone != "" {
			location, lErr := time.LoadLocation(app.BaseConfig.TimeZone)

			if lErr != nil {
				log.Fatalf(lErr.Error())
			}

			app.Location = location
		}
	}

	// Инициализация трассировки
	{
		tErr := app.initTraceClient()

		if tErr != nil {
			log.Fatalf(tErr.Error())
		}
	}

	// Инициализация сентри
	{
		if err := sentry.Init(sentry.ClientOptions{
			AttachStacktrace: true,
			TracesSampleRate: 1.0,
			Dsn:              app.BaseConfig.SentryDsn,
		}); err != nil {
			fmt.Printf("Sentry initialization failed: %v\n", err)
		}
	}

	if app.PrepareComponentsFunc == nil {
		log.Fatalf("PrepareComponents not defined")
	}

	if err := app.PrepareComponentsFunc(app); err != nil {
		log.Fatalf("PrepareComponents error: %v", err)
	}
}

// AddStopHook Добавить функцию, которая будет вызвана на shutdown
func (app *App) AddStopHook(hook func(ctx context.Context) error) {
	app.stopHooks = append(app.stopHooks, hook)
}

// waitForShutdown Метод для красивой остановки приложения
func (app *App) waitForShutdown() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	<-stop // Ожидание сигнала

	log.Println("Shutting down application...")

	ctx, cancel := context.WithTimeout(context.Background(), app.shutdownTimeout)
	defer cancel()

	for _, hook := range app.stopHooks {
		if err := hook(ctx); err != nil {
			log.Printf("Shutdown hook error: %v", err)
		}
	}

	log.Println("Application stopped gracefully.")
}

// RunHttp Запуск веб сервера
func (app *App) RunHttp() error {
	once.Do(func() {
		app.initApp()
	})

	// Инициализаця метрик
	{
		app.initHttpMetrics()
	}

	//инициализация ginHelpers
	app.Router = ginHelper.InitRouter(app.BaseConfig)

	if app.PrepareHttpFunc == nil {
		return errors.New("apps PrepareHttpFunc not defined")
	}

	if err := app.PrepareHttpFunc(app); err != nil {
		return err
	}

	//запускаем сервер
	gErr := app.Router.Run(app.BaseConfig.ServerAddress)

	if gErr != nil {
		return gErr
	}

	return nil
}

// RunConsumer запуск консьюмера
func (app *App) RunConsumer() error {
	once.Do(func() {
		app.initApp()
	})

	if app.PrepareConsumerFunc == nil {
		return errors.New("apps PrepareConsumerFunc not defined")
	}

	if err := app.PrepareConsumerFunc(app); err != nil {
		return err
	}

	if app.AmqpClient != nil {
		if err := app.AmqpClient.Run(); err != nil {
			return err
		}
	}

	return nil
}
