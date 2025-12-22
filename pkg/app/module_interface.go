package app

import "context"

type ModuleInterface interface {
	Name() string
	Register(a *App) error
	Start(a *App) error
	Stop(ctx context.Context) error
}
