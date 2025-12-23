package app

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type KernelManager struct {
	mu      sync.Mutex
	kernels map[string]KernelInterface
	states  map[string]*kernelState
}

type kernelState struct {
	// Init
	initOnce sync.Once
	initErr  error
	initDone chan struct{} // закрывается когда init завершён

	// Start
	startOnce sync.Once
	startErr  error
	startDone chan struct{} // закрывается когда start завершён

	// Stop hook (чтобы не добавить дважды)
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

// Init выполняет Init(kernel) ровно один раз, остальные ждут завершения и получают ту же ошибку.
func (km *KernelManager) Init(app *App, name string) error {
	k, st, err := km.get(name)
	if err != nil {
		return err
	}

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
	if err := km.Init(app, name); err != nil {
		return err
	}

	k, st, err := km.get(name)
	if err != nil {
		return err
	}

	st.startOnce.Do(func() {
		defer close(st.startDone)

		if err := k.Start(app); err != nil {
			// попытка корректно остановить, если старт провалился
			ctx, cancel := context.WithTimeout(app.ctx, app.shutdownTimeout)
			defer cancel()
			_ = k.Stop(ctx)

			st.startErr = err
			return
		}

		// Регистрируем stop-hook строго один раз
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

	// Повторный Run -> просто nil (идемпотентно)
	return nil
}
