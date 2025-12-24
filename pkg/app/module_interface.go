package app

type ModuleInterface interface {
	Name() string
	Init(a *App) error
}
