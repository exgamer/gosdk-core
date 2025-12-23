package app

type ModuleInterface interface {
	Name() string
	Register(a *App) error
}
