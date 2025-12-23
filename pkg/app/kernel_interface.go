package app

import "context"

type KernelInterface interface {
	Name() string
	Init(a *App) error
	Start(a *App) error
	Stop(ctx context.Context) error
}
