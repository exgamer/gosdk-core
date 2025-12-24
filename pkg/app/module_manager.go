package app

import (
	"errors"
	"fmt"
	"sync"
)

var ErrModuleNotInited = errors.New("module not initialized")

type ModuleManager struct {
	mu      sync.RWMutex
	modules map[string]ModuleInterface
	states  map[string]*moduleState
	order   []string
}

type moduleState struct {
	initOnce sync.Once
	initErr  error
	initDone chan struct{}
}

func NewModuleManager() *ModuleManager {
	return &ModuleManager{
		modules: make(map[string]ModuleInterface),
		states:  make(map[string]*moduleState),
		order:   make([]string, 0, 8),
	}
}

func (m *ModuleManager) RegisterAll(modules ...ModuleInterface) error {
	for _, module := range modules {
		if err := m.Register(module); err != nil {
			return err
		}
	}
	return nil
}

func (m *ModuleManager) Register(mod ModuleInterface) error {
	if mod == nil {
		return errors.New("module is nil")
	}

	name := mod.Name()
	if name == "" {
		return errors.New("module name is empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.modules[name]; exists {
		return fmt.Errorf("module already registered: %s", name)
	}

	m.modules[name] = mod
	m.states[name] = &moduleState{initDone: make(chan struct{})}
	m.order = append(m.order, name)

	return nil
}

func (m *ModuleManager) get(name string) (ModuleInterface, *moduleState, error) {
	m.mu.RLock()
	mod, ok := m.modules[name]
	st := m.states[name]
	m.mu.RUnlock()

	if !ok || mod == nil {
		return nil, nil, fmt.Errorf("module not registered: %s", name)
	}

	// страховка: если st по какой-то причине nil (не должно быть после Register)
	if st == nil {
		m.mu.Lock()
		st = m.states[name]
		if st == nil {
			st = &moduleState{initDone: make(chan struct{})}
			m.states[name] = st
		}
		m.mu.Unlock()
	}

	return mod, st, nil
}

func (m *ModuleManager) InitAll(app *App) error {
	m.mu.RLock()
	order := append([]string(nil), m.order...)
	m.mu.RUnlock()

	for _, name := range order {
		if err := m.Init(app, name); err != nil {
			return err
		}
	}
	return nil
}

func (m *ModuleManager) Init(app *App, name string) error {
	mod, st, err := m.get(name)
	if err != nil {
		return err
	}

	st.initOnce.Do(func() {
		defer close(st.initDone)
		st.initErr = mod.Init(app)
	})

	<-st.initDone

	if st.initErr != nil {
		return fmt.Errorf("init %s: %w", name, st.initErr)
	}
	return nil
}
