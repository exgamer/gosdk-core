package app

import "context"

type ModuleInterface interface {
	Name() string
	Register(a *App) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}
