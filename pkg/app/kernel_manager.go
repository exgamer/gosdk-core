package app

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

var ErrKernelNotInited = errors.New("kernel not initialized")

type KernelManager struct {
	mu      sync.Mutex
	kernels map[string]KernelInterface
	states  map[string]*kernelState
}

type kernelState struct {
	// Init
	initOnce   sync.Once
	initErr    error
	initDone   chan struct{}
	initCalled bool // <-- добавили

	// Start
	startOnce sync.Once
	startErr  error
	startDone chan struct{}

	stopHookOnce sync.Once
}

func NewKernelManager() *KernelManager {
	return &KernelManager{
		kernels: make(map[string]KernelInterface),
		states:  make(map[string]*kernelState),
	}
}

func (km *KernelManager) Register(k KernelInterface) error {
	if k == nil {
		return errors.New("kernel is nil")
	}

	name := k.Name()
	if name == "" {
		return errors.New("kernel name is empty")
	}

	km.mu.Lock()
	defer km.mu.Unlock()

	if _, exists := km.kernels[name]; exists {
		return fmt.Errorf("kernel already registered: %s", name)
	}

	km.kernels[name] = k
	km.states[name] = &kernelState{
		initDone:  make(chan struct{}),
		startDone: make(chan struct{}),
	}

	return nil
}

func (km *KernelManager) get(name string) (KernelInterface, *kernelState, error) {
	km.mu.Lock()
	defer km.mu.Unlock()

	k, ok := km.kernels[name]
	if !ok || k == nil {
		return nil, nil, fmt.Errorf("kernel not registered: %s", name)
	}
	st := km.states[name]
	if st == nil {
		// на всякий случай
		st = &kernelState{
			initDone:  make(chan struct{}),
			startDone: make(chan struct{}),
		}
		km.states[name] = st
	}
	return k, st, nil
}

func (km *KernelManager) InitAll(app *App) error {
	for name, _ := range km.kernels {
		err := km.Init(app, name)
		if err != nil {
			return err
		}
	}

	return nil
}

// Init выполняет Init(kernel) ровно один раз, остальные ждут завершения и получают ту же ошибку.
func (km *KernelManager) Init(app *App, name string) error {
	k, st, err := km.get(name)
	if err != nil {
		return err
	}

	// помечаем, что Init был инициирован хотя бы раз
	km.mu.Lock()
	if !st.initCalled {
		st.initCalled = true
	}
	km.mu.Unlock()

	st.initOnce.Do(func() {
		defer close(st.initDone)
		st.initErr = k.Init(app)
	})

	<-st.initDone

	if st.initErr != nil {
		return fmt.Errorf("init %s: %w", name, st.initErr)
	}

	return nil
}

// Run гарантирует Init + Start (start тоже один раз, остальные ждут).
func (km *KernelManager) Run(app *App, name string) error {
	k, st, err := km.get(name)
	if err != nil {
		return err
	}

	km.mu.Lock()
	initCalled := st.initCalled
	km.mu.Unlock()

	if !initCalled {
		return fmt.Errorf("%w: %s (call Init first)", ErrKernelNotInited, name)
	}

	// ждём завершения init, если он был запущен
	<-st.initDone
	if st.initErr != nil {
		return fmt.Errorf("init %s: %w", name, st.initErr)
	}

	st.startOnce.Do(func() {
		defer close(st.startDone)

		if err := k.Start(app); err != nil {
			ctx, cancel := context.WithTimeout(app.ctx, app.shutdownTimeout)
			defer cancel()
			_ = k.Stop(ctx)

			st.startErr = err
			return
		}

		st.stopHookOnce.Do(func() {
			app.AddStopHook(func(ctx context.Context) error {
				return k.Stop(ctx)
			})
		})
	})

	<-st.startDone
	if st.startErr != nil {
		return fmt.Errorf("start %s: %w", name, st.startErr)
	}

	return nil
}
