package di

import (
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type service interface{ Name() string }

type serviceImpl struct{ name string }

func (s *serviceImpl) Name() string { return s.name }

type valueService struct{ name string }

func (s valueService) Name() string { return s.name }

type testConfig struct{ Addr string }

type testConfigAlias = testConfig

type port int

type box[T any] struct{ Value T }

type configError struct{ field string }

func (e *configError) Error() string { return "invalid config field " + e.field }

func newTestConfig() (*testConfig, error) {
	return &testConfig{Addr: "constructed"}, nil
}

// mustResolve - резолвит зависимость и валит тест при ошибке
func mustResolve[T any](t *testing.T, c *Container) T {
	t.Helper()

	v, err := Resolve[T](c)
	if err != nil {
		t.Fatalf("resolve %T: %v", v, err)
	}

	return v
}

// expectErr - проверяет, что Resolve вернул ожидаемую ошибку и zero value
func expectErr[T any](t *testing.T, c *Container, target error) {
	t.Helper()

	v, err := Resolve[T](c)
	if !errors.Is(err, target) {
		t.Fatalf("expected %v, got %v", target, err)
	}

	var zero T
	if any(v) != any(zero) {
		t.Fatalf("expected zero value, got %v", v)
	}
}

// expectPanic - проверяет, что fn паникует с сообщением, содержащим substr
func expectPanic(t *testing.T, substr string, fn func()) {
	t.Helper()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic")
		}
		if msg, _ := r.(string); !strings.Contains(msg, substr) {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()

	fn()
}

func TestRegisterInstance(t *testing.T) {
	t.Run("pointer to struct", func(t *testing.T) {
		c := NewContainer()
		cfg := &testConfig{Addr: "localhost"}
		Register(c, cfg)

		for i := 0; i < 3; i++ {
			if got := mustResolve[*testConfig](t, c); got != cfg {
				t.Fatalf("resolve #%d: expected same pointer", i+1)
			}
		}
	})

	t.Run("struct value", func(t *testing.T) {
		c := NewContainer()
		Register(c, testConfig{Addr: "localhost"})

		if got := mustResolve[testConfig](t, c); got.Addr != "localhost" {
			t.Fatalf("unexpected addr %q", got.Addr)
		}
	})

	t.Run("struct value is copied on resolve", func(t *testing.T) {
		c := NewContainer()
		Register(c, testConfig{Addr: "localhost"})

		got := mustResolve[testConfig](t, c)
		got.Addr = "changed"

		if again := mustResolve[testConfig](t, c); again.Addr != "localhost" {
			t.Fatalf("container value mutated: %q", again.Addr)
		}
	})

	t.Run("pointer is shared on resolve", func(t *testing.T) {
		c := NewContainer()
		Register(c, &testConfig{Addr: "localhost"})

		mustResolve[*testConfig](t, c).Addr = "changed"

		if again := mustResolve[*testConfig](t, c); again.Addr != "changed" {
			t.Fatalf("expected shared pointer, got %q", again.Addr)
		}
	})

	t.Run("primitives", func(t *testing.T) {
		c := NewContainer()
		Register(c, 42)
		Register(c, "str")
		Register(c, true)
		Register(c, 3.14)

		if mustResolve[int](t, c) != 42 {
			t.Fatal("int")
		}
		if mustResolve[string](t, c) != "str" {
			t.Fatal("string")
		}
		if !mustResolve[bool](t, c) {
			t.Fatal("bool")
		}
		if mustResolve[float64](t, c) != 3.14 {
			t.Fatal("float64")
		}
	})

	t.Run("slice and map", func(t *testing.T) {
		c := NewContainer()
		Register(c, []string{"a", "b"})
		Register(c, map[string]int{"a": 1})

		if got := mustResolve[[]string](t, c); len(got) != 2 || got[1] != "b" {
			t.Fatalf("unexpected slice %v", got)
		}
		if got := mustResolve[map[string]int](t, c); got["a"] != 1 {
			t.Fatalf("unexpected map %v", got)
		}
	})

	t.Run("channel", func(t *testing.T) {
		c := NewContainer()
		ch := make(chan int, 1)
		Register(c, ch)

		if mustResolve[chan int](t, c) != ch {
			t.Fatal("expected same channel")
		}
		expectErr[<-chan int](t, c, ErrDependencyNotFound)
	})

	t.Run("named type differs from underlying", func(t *testing.T) {
		c := NewContainer()
		Register(c, port(8080))

		if mustResolve[port](t, c) != 8080 {
			t.Fatal("port")
		}
		expectErr[int](t, c, ErrDependencyNotFound)
	})

	t.Run("type alias is the same type", func(t *testing.T) {
		c := NewContainer()
		Register(c, &testConfig{Addr: "localhost"})

		if got := mustResolve[*testConfigAlias](t, c); got.Addr != "localhost" {
			t.Fatalf("unexpected addr %q", got.Addr)
		}
	})

	t.Run("generic instantiations are different types", func(t *testing.T) {
		c := NewContainer()
		Register(c, box[int]{Value: 1})
		Register(c, box[string]{Value: "s"})

		if mustResolve[box[int]](t, c).Value != 1 {
			t.Fatal("box[int]")
		}
		if mustResolve[box[string]](t, c).Value != "s" {
			t.Fatal("box[string]")
		}
	})

	t.Run("anonymous struct", func(t *testing.T) {
		c := NewContainer()
		Register(c, struct{ A int }{A: 7})

		if mustResolve[struct{ A int }](t, c).A != 7 {
			t.Fatal("anonymous struct")
		}
	})

	t.Run("pointer and value are different keys", func(t *testing.T) {
		c := NewContainer()
		Register(c, testConfig{Addr: "value"})
		Register(c, &testConfig{Addr: "pointer"})

		if mustResolve[testConfig](t, c).Addr != "value" {
			t.Fatal("value")
		}
		if mustResolve[*testConfig](t, c).Addr != "pointer" {
			t.Fatal("pointer")
		}
	})

	t.Run("pointer registered, value not found", func(t *testing.T) {
		c := NewContainer()
		Register(c, &testConfig{})

		expectErr[testConfig](t, c, ErrDependencyNotFound)
	})

	t.Run("re-register overwrites", func(t *testing.T) {
		c := NewContainer()
		Register(c, &testConfig{Addr: "first"})
		Register(c, &testConfig{Addr: "second"})

		if got := mustResolve[*testConfig](t, c); got.Addr != "second" {
			t.Fatalf("expected last registration, got %q", got.Addr)
		}
	})

	t.Run("typed nil pointer", func(t *testing.T) {
		c := NewContainer()
		var cfg *testConfig
		Register(c, cfg)

		if got := mustResolve[*testConfig](t, c); got != nil {
			t.Fatal("expected nil")
		}
	})
}

func TestRegisterInterface(t *testing.T) {
	t.Run("interface-typed variable resolves by interface", func(t *testing.T) {
		c := NewContainer()
		var s service = &serviceImpl{name: "svc"}
		Register(c, s)

		if got := mustResolve[service](t, c); got.Name() != "svc" {
			t.Fatalf("unexpected name %q", got.Name())
		}
	})

	t.Run("explicit type parameter", func(t *testing.T) {
		c := NewContainer()
		Register[service](c, &serviceImpl{name: "svc"})

		if got := mustResolve[service](t, c); got.Name() != "svc" {
			t.Fatalf("unexpected name %q", got.Name())
		}
	})

	t.Run("value receiver implementation", func(t *testing.T) {
		c := NewContainer()
		Register[service](c, valueService{name: "svc"})

		if got := mustResolve[service](t, c); got.Name() != "svc" {
			t.Fatalf("unexpected name %q", got.Name())
		}
	})

	t.Run("registered as interface, concrete not found", func(t *testing.T) {
		c := NewContainer()
		Register[service](c, &serviceImpl{})

		expectErr[*serviceImpl](t, c, ErrDependencyNotFound)
	})

	t.Run("registered as concrete, interface not found", func(t *testing.T) {
		c := NewContainer()
		Register(c, &serviceImpl{})

		expectErr[service](t, c, ErrDependencyNotFound)
	})

	t.Run("interface and concrete registered side by side", func(t *testing.T) {
		c := NewContainer()
		impl := &serviceImpl{name: "concrete"}
		Register(c, impl)
		Register[service](c, &serviceImpl{name: "interface"})

		if mustResolve[*serviceImpl](t, c) != impl {
			t.Fatal("concrete")
		}
		if mustResolve[service](t, c).Name() != "interface" {
			t.Fatal("interface")
		}
	})

	t.Run("nil interface", func(t *testing.T) {
		c := NewContainer()
		var s service
		Register(c, s)

		for i := 0; i < 2; i++ {
			if got := mustResolve[service](t, c); got != nil {
				t.Fatalf("resolve #%d: expected nil", i+1)
			}
		}
	})

	t.Run("error interface", func(t *testing.T) {
		c := NewContainer()
		sentinel := errors.New("boom")
		Register[error](c, sentinel)

		if mustResolve[error](t, c) != sentinel {
			t.Fatal("expected same error")
		}
	})

	t.Run("empty interface", func(t *testing.T) {
		c := NewContainer()
		Register[any](c, 5)

		if got := mustResolve[any](t, c); got != 5 {
			t.Fatalf("unexpected value %v", got)
		}
		expectErr[int](t, c, ErrDependencyNotFound)
	})
}

func TestFactory(t *testing.T) {
	t.Run("is lazy", func(t *testing.T) {
		c := NewContainer()
		var calls int
		Register(c, func() *testConfig {
			calls++

			return &testConfig{}
		})

		if calls != 0 {
			t.Fatal("factory called on register")
		}

		mustResolve[*testConfig](t, c)

		if calls != 1 {
			t.Fatalf("factory called %d times, want 1", calls)
		}
	})

	t.Run("pointer singleton", func(t *testing.T) {
		c := NewContainer()
		var calls int
		Register(c, func() *testConfig {
			calls++

			return &testConfig{Addr: "localhost"}
		})

		first := mustResolve[*testConfig](t, c)
		for i := 2; i <= 3; i++ {
			if mustResolve[*testConfig](t, c) != first {
				t.Fatalf("resolve #%d: expected singleton", i)
			}
		}
		if calls != 1 {
			t.Fatalf("factory called %d times, want 1", calls)
		}
	})

	t.Run("interface singleton", func(t *testing.T) {
		c := NewContainer()
		var calls int
		Register(c, func() service {
			calls++

			return &serviceImpl{name: "svc"}
		})

		first := mustResolve[service](t, c)
		for i := 2; i <= 3; i++ {
			if mustResolve[service](t, c) != first {
				t.Fatalf("resolve #%d: expected singleton", i)
			}
		}
		if calls != 1 {
			t.Fatalf("factory called %d times, want 1", calls)
		}
	})

	t.Run("struct value", func(t *testing.T) {
		c := NewContainer()
		Register(c, func() testConfig { return testConfig{Addr: "localhost"} })

		for i := 0; i < 2; i++ {
			if got := mustResolve[testConfig](t, c); got.Addr != "localhost" {
				t.Fatalf("resolve #%d: unexpected addr %q", i+1, got.Addr)
			}
		}
	})

	t.Run("primitive", func(t *testing.T) {
		c := NewContainer()
		Register(c, func() int { return 42 })

		if mustResolve[int](t, c) != 42 {
			t.Fatal("int")
		}
	})

	t.Run("returns nil interface", func(t *testing.T) {
		c := NewContainer()
		var calls int
		Register(c, func() service {
			calls++

			return nil
		})

		for i := 0; i < 2; i++ {
			if got := mustResolve[service](t, c); got != nil {
				t.Fatalf("resolve #%d: expected nil", i+1)
			}
		}
		if calls != 1 {
			t.Fatalf("factory called %d times, want 1", calls)
		}
	})

	t.Run("returns nil pointer", func(t *testing.T) {
		c := NewContainer()
		Register(c, func() *testConfig { return nil })

		if got := mustResolve[*testConfig](t, c); got != nil {
			t.Fatal("expected nil")
		}
	})

	t.Run("returns func", func(t *testing.T) {
		c := NewContainer()
		Register(c, func() func() int { return func() int { return 7 } })

		if mustResolve[func() int](t, c)() != 7 {
			t.Fatal("func")
		}
	})

	t.Run("keyed by declared return type", func(t *testing.T) {
		c := NewContainer()
		Register(c, func() *serviceImpl { return &serviceImpl{} })

		mustResolve[*serviceImpl](t, c)
		expectErr[service](t, c, ErrDependencyNotFound)
	})

	t.Run("resolves dependencies inside", func(t *testing.T) {
		c := NewContainer()
		Register(c, &testConfig{Addr: "localhost"})
		Register(c, func() service {
			return &serviceImpl{name: mustResolve[*testConfig](t, c).Addr}
		})

		if got := mustResolve[service](t, c); got.Name() != "localhost" {
			t.Fatalf("unexpected name %q", got.Name())
		}
	})

	t.Run("chain of factories", func(t *testing.T) {
		c := NewContainer()
		Register(c, func() *testConfig { return &testConfig{Addr: "chained"} })
		Register(c, func() service {
			return &serviceImpl{name: mustResolve[*testConfig](t, c).Addr}
		})
		Register(c, func() box[service] {
			return box[service]{Value: mustResolve[service](t, c)}
		})

		if got := mustResolve[box[service]](t, c); got.Value.Name() != "chained" {
			t.Fatalf("unexpected name %q", got.Value.Name())
		}
	})

	t.Run("factory re-registered after resolve replaces instance", func(t *testing.T) {
		c := NewContainer()
		Register(c, func() *testConfig { return &testConfig{Addr: "first"} })
		mustResolve[*testConfig](t, c)

		Register(c, func() *testConfig { return &testConfig{Addr: "second"} })

		if got := mustResolve[*testConfig](t, c); got.Addr != "second" {
			t.Fatalf("expected new factory result, got %q", got.Addr)
		}
	})

	t.Run("re-register before resolve overwrites", func(t *testing.T) {
		c := NewContainer()
		Register(c, func() *testConfig { return &testConfig{Addr: "first"} })
		Register(c, func() *testConfig { return &testConfig{Addr: "second"} })

		if got := mustResolve[*testConfig](t, c); got.Addr != "second" {
			t.Fatalf("expected last factory, got %q", got.Addr)
		}
	})

	t.Run("panic keeps factory for retry", func(t *testing.T) {
		c := NewContainer()
		var calls int
		Register(c, func() *testConfig {
			calls++
			if calls == 1 {
				panic("first call fails")
			}

			return &testConfig{Addr: "retry"}
		})

		expectPanic(t, "first call fails", func() { _, _ = Resolve[*testConfig](c) })

		if got := mustResolve[*testConfig](t, c); got.Addr != "retry" {
			t.Fatalf("unexpected addr %q", got.Addr)
		}
	})

	t.Run("named func type", func(t *testing.T) {
		type configFactory func() *testConfig

		c := NewContainer()
		Register(c, configFactory(func() *testConfig { return &testConfig{Addr: "named"} }))

		if got := mustResolve[*testConfig](t, c); got.Addr != "named" {
			t.Fatalf("unexpected addr %q", got.Addr)
		}
	})
}

func TestFactoryWithError(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		c := NewContainer()
		var calls int
		Register(c, func() (*testConfig, error) {
			calls++

			return &testConfig{Addr: "localhost"}, nil
		})

		first := mustResolve[*testConfig](t, c)
		if first.Addr != "localhost" {
			t.Fatalf("unexpected addr %q", first.Addr)
		}
		if mustResolve[*testConfig](t, c) != first {
			t.Fatal("expected singleton")
		}
		if calls != 1 {
			t.Fatalf("factory called %d times, want 1", calls)
		}
	})

	t.Run("constructor function", func(t *testing.T) {
		c := NewContainer()
		Register(c, newTestConfig)

		if got := mustResolve[*testConfig](t, c); got.Addr != "constructed" {
			t.Fatalf("unexpected addr %q", got.Addr)
		}
	})

	t.Run("interface", func(t *testing.T) {
		c := NewContainer()
		Register(c, func() (service, error) { return &serviceImpl{name: "svc"}, nil })

		if got := mustResolve[service](t, c); got.Name() != "svc" {
			t.Fatalf("unexpected name %q", got.Name())
		}
	})

	t.Run("error is returned and wrapped", func(t *testing.T) {
		c := NewContainer()
		cause := errors.New("connection refused")
		Register(c, func() (*testConfig, error) { return nil, cause })

		_, err := Resolve[*testConfig](c)
		if !errors.Is(err, ErrFactoryFailed) {
			t.Fatalf("expected ErrFactoryFailed, got %v", err)
		}
		if !errors.Is(err, cause) {
			t.Fatalf("expected cause in chain, got %v", err)
		}
		if err.Error() != "factory failed: *di.testConfig: connection refused" {
			t.Fatalf("unexpected message %q", err.Error())
		}
	})

	t.Run("custom error type is reachable via errors.As", func(t *testing.T) {
		c := NewContainer()
		Register(c, func() (*testConfig, error) { return nil, &configError{field: "Addr"} })

		_, err := Resolve[*testConfig](c)

		var cfgErr *configError
		if !errors.As(err, &cfgErr) || cfgErr.field != "Addr" {
			t.Fatalf("expected configError, got %v", err)
		}
	})

	t.Run("zero value on error even if factory returned value", func(t *testing.T) {
		c := NewContainer()
		Register(c, func() (*testConfig, error) { return &testConfig{}, errors.New("boom") })

		expectErr[*testConfig](t, c, ErrFactoryFailed)
	})

	t.Run("error is not cached, retry succeeds", func(t *testing.T) {
		c := NewContainer()
		var calls int
		Register(c, func() (*testConfig, error) {
			calls++
			if calls == 1 {
				return nil, errors.New("not ready")
			}

			return &testConfig{Addr: "ready"}, nil
		})

		expectErr[*testConfig](t, c, ErrFactoryFailed)

		first := mustResolve[*testConfig](t, c)
		if first.Addr != "ready" {
			t.Fatalf("unexpected addr %q", first.Addr)
		}
		if mustResolve[*testConfig](t, c) != first {
			t.Fatal("expected singleton after success")
		}
		if calls != 2 {
			t.Fatalf("factory called %d times, want 2", calls)
		}
	})

	t.Run("nil value without error", func(t *testing.T) {
		c := NewContainer()
		Register(c, func() (service, error) { return nil, nil })

		if got := mustResolve[service](t, c); got != nil {
			t.Fatal("expected nil")
		}
	})

	t.Run("propagates error of nested dependency", func(t *testing.T) {
		c := NewContainer()
		cause := errors.New("db down")
		Register(c, func() (*testConfig, error) { return nil, cause })
		Register(c, func() (service, error) {
			cfg, err := Resolve[*testConfig](c)
			if err != nil {
				return nil, err
			}

			return &serviceImpl{name: cfg.Addr}, nil
		})

		_, err := Resolve[service](c)
		if !errors.Is(err, cause) {
			t.Fatalf("expected nested cause, got %v", err)
		}
		if err.Error() != "factory failed: di.service: factory failed: *di.testConfig: db down" {
			t.Fatalf("unexpected message %q", err.Error())
		}
	})
}

func TestFactoryInvalidSignature(t *testing.T) {
	cases := map[string]func(c *Container){
		"no return value":         func(c *Container) { Register(c, func() {}) },
		"with argument":           func(c *Container) { Register(c, func(string) *testConfig { return nil }) },
		"variadic":                func(c *Container) { Register(c, func(...int) *testConfig { return nil }) },
		"argument and error":      func(c *Container) { Register(c, func(int) (*testConfig, error) { return nil, nil }) },
		"second result not error": func(c *Container) { Register(c, func() (*testConfig, bool) { return nil, false }) },
		"three results":           func(c *Container) { Register(c, func() (int, int, error) { return 0, 0, nil }) },
		"error-like but not error": func(c *Container) {
			Register(c, func() (*testConfig, *configError) { return nil, nil })
		},
	}

	for name, register := range cases {
		t.Run(name, func(t *testing.T) {
			c := NewContainer()

			expectPanic(t, "invalid factory", func() { register(c) })
		})
	}

	t.Run("nil factory", func(t *testing.T) {
		c := NewContainer()
		var create func() *testConfig

		expectPanic(t, "nil factory", func() { Register(c, create) })
	})

	t.Run("invalid factory is not registered", func(t *testing.T) {
		c := NewContainer()

		expectPanic(t, "invalid factory", func() {
			Register(c, func(string) *testConfig { return nil })
		})
		expectErr[*testConfig](t, c, ErrDependencyNotFound)
	})
}

func TestInstanceAndFactoryPriority(t *testing.T) {
	t.Run("instance registered after factory wins", func(t *testing.T) {
		c := NewContainer()
		var calls int
		Register(c, func() *testConfig {
			calls++

			return &testConfig{Addr: "factory"}
		})
		Register(c, &testConfig{Addr: "instance"})

		if got := mustResolve[*testConfig](t, c); got.Addr != "instance" {
			t.Fatalf("expected instance, got %q", got.Addr)
		}
		if calls != 0 {
			t.Fatal("factory must not be called")
		}
	})

	t.Run("factory registered after instance wins", func(t *testing.T) {
		c := NewContainer()
		Register(c, &testConfig{Addr: "instance"})
		Register(c, func() *testConfig { return &testConfig{Addr: "factory"} })

		if got := mustResolve[*testConfig](t, c); got.Addr != "factory" {
			t.Fatalf("expected factory, got %q", got.Addr)
		}
	})

	t.Run("instance registered after factory removes factory", func(t *testing.T) {
		c := NewContainer()
		Register(c, func() *testConfig { return &testConfig{} })
		Register(c, &testConfig{Addr: "instance"})

		if len(c.functions) != 0 {
			t.Fatal("factory must be removed")
		}
	})

	t.Run("instance registered while factory is running wins", func(t *testing.T) {
		c := NewContainer()
		started, release := make(chan struct{}), make(chan struct{})
		Register(c, func() *testConfig {
			close(started)
			<-release

			return &testConfig{Addr: "factory"}
		})

		result := make(chan *testConfig)
		go func() { result <- mustResolve[*testConfig](t, c) }()

		<-started
		Register(c, &testConfig{Addr: "instance"})
		close(release)

		if got := <-result; got.Addr != "instance" {
			t.Fatalf("builder: expected instance, got %q", got.Addr)
		}
		if got := mustResolve[*testConfig](t, c); got.Addr != "instance" {
			t.Fatalf("expected instance, got %q", got.Addr)
		}
	})

	t.Run("factory registered while old factory is running wins", func(t *testing.T) {
		c := NewContainer()
		started, release := make(chan struct{}), make(chan struct{})
		Register(c, func() *testConfig {
			close(started)
			<-release

			return &testConfig{Addr: "old"}
		})

		result := make(chan *testConfig)
		go func() { result <- mustResolve[*testConfig](t, c) }()

		<-started
		Register(c, func() *testConfig { return &testConfig{Addr: "new"} })
		close(release)

		if got := <-result; got.Addr != "old" {
			t.Fatalf("builder: expected its own result, got %q", got.Addr)
		}
		if got := mustResolve[*testConfig](t, c); got.Addr != "new" {
			t.Fatalf("expected new factory result, got %q", got.Addr)
		}
	})

	t.Run("instance overwrites resolved factory result", func(t *testing.T) {
		c := NewContainer()
		Register(c, func() *testConfig { return &testConfig{Addr: "factory"} })
		mustResolve[*testConfig](t, c)

		Register(c, &testConfig{Addr: "instance"})

		if got := mustResolve[*testConfig](t, c); got.Addr != "instance" {
			t.Fatalf("expected instance, got %q", got.Addr)
		}
	})
}

func TestResolveErrors(t *testing.T) {
	t.Run("empty container", func(t *testing.T) {
		expectErr[*testConfig](t, NewContainer(), ErrDependencyNotFound)
		expectErr[service](t, NewContainer(), ErrDependencyNotFound)
		expectErr[int](t, NewContainer(), ErrDependencyNotFound)
	})

	t.Run("not found message is backward compatible", func(t *testing.T) {
		_, err := Resolve[*testConfig](NewContainer())

		if err.Error() != "dependency not found: *di.testConfig" {
			t.Fatalf("unexpected message %q", err.Error())
		}
	})

	t.Run("errors are distinguishable", func(t *testing.T) {
		_, err := Resolve[*testConfig](NewContainer())

		if errors.Is(err, ErrFactoryFailed) {
			t.Fatal("not found must not match factory failed")
		}
	})
}

func TestContainersAreIsolated(t *testing.T) {
	a, b := NewContainer(), NewContainer()
	Register(a, &testConfig{Addr: "a"})
	Register(b, func() *testConfig { return &testConfig{Addr: "b"} })

	if mustResolve[*testConfig](t, a).Addr != "a" {
		t.Fatal("container a")
	}
	if mustResolve[*testConfig](t, b).Addr != "b" {
		t.Fatal("container b")
	}
	expectErr[service](t, a, ErrDependencyNotFound)
}

func TestConcurrency(t *testing.T) {
	const n = 100

	t.Run("factory singleton", func(t *testing.T) {
		c := NewContainer()
		var calls atomic.Int32
		defer func() {
			if got := calls.Load(); got != 1 {
				t.Fatalf("factory called %d times, want exactly 1", got)
			}
		}()
		Register(c, func() service {
			calls.Add(1)
			time.Sleep(time.Millisecond) // расширяем окно гонки

			return &serviceImpl{name: "svc"}
		})

		results := make([]service, n)
		start := make(chan struct{})
		var wg sync.WaitGroup

		for i := 0; i < n; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				<-start

				s, err := Resolve[service](c)
				if err != nil {
					t.Errorf("resolve: %v", err)

					return
				}
				results[i] = s
			}(i)
		}
		close(start)
		wg.Wait()

		for i := 1; i < n; i++ {
			if results[i] != results[0] {
				t.Fatalf("goroutine %d got a different instance", i)
			}
		}

		if mustResolve[service](t, c) != results[0] {
			t.Fatal("cached instance differs")
		}
	})

	t.Run("register and resolve different types", func(t *testing.T) {
		c := NewContainer()
		var wg sync.WaitGroup

		for i := 0; i < n; i++ {
			wg.Add(3)
			go func() {
				defer wg.Done()
				Register(c, &testConfig{})
			}()
			go func() {
				defer wg.Done()
				Register(c, func() service { return &serviceImpl{} })
			}()
			go func() {
				defer wg.Done()
				_, _ = Resolve[*testConfig](c)
				_, _ = Resolve[service](c)
			}()
		}
		wg.Wait()

		mustResolve[*testConfig](t, c)
		mustResolve[service](t, c)
	})

	t.Run("factory resolving dependencies does not deadlock", func(t *testing.T) {
		c := NewContainer()
		var factoryCalls atomic.Int32
		Register(c, func() *testConfig { return &testConfig{Addr: "dep"} })
		Register(c, func() service {
			factoryCalls.Add(1)

			return &serviceImpl{name: mustResolve[*testConfig](t, c).Addr}
		})

		done := make(chan struct{})
		go func() {
			defer close(done)

			var wg sync.WaitGroup
			for i := 0; i < n; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					_, _ = Resolve[service](c)
				}()
			}
			wg.Wait()
		}()

		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("deadlock")
		}

		if mustResolve[service](t, c).Name() != "dep" {
			t.Fatal("unexpected name")
		}
		if factoryCalls.Load() == 0 {
			t.Fatal("factory was not called")
		}
	})
}

func TestConcurrentFactoryFailures(t *testing.T) {
	const n = 50

	t.Run("error is shared with waiters and factory is called once", func(t *testing.T) {
		c := NewContainer()
		var calls atomic.Int32
		cause := errors.New("db down")
		Register(c, func() (*testConfig, error) {
			calls.Add(1)
			time.Sleep(5 * time.Millisecond)

			return nil, cause
		})

		errs := resolveConcurrently[*testConfig](c, n)

		for i, err := range errs {
			if !errors.Is(err, cause) {
				t.Fatalf("goroutine %d: expected cause, got %v", i, err)
			}
		}
		if got := calls.Load(); got != 1 {
			t.Fatalf("factory called %d times, want 1", got)
		}
	})

	t.Run("retry after shared error", func(t *testing.T) {
		c := NewContainer()
		var calls atomic.Int32
		Register(c, func() (*testConfig, error) {
			if calls.Add(1) == 1 {
				time.Sleep(5 * time.Millisecond)

				return nil, errors.New("not ready")
			}

			return &testConfig{Addr: "ready"}, nil
		})

		resolveConcurrently[*testConfig](c, n)

		if got := mustResolve[*testConfig](t, c); got.Addr != "ready" {
			t.Fatalf("unexpected addr %q", got.Addr)
		}
	})

	t.Run("panic releases waiters with error", func(t *testing.T) {
		c := NewContainer()
		started, release := make(chan struct{}), make(chan struct{})
		var calls atomic.Int32
		Register(c, func() *testConfig {
			if calls.Add(1) == 1 {
				close(started)
				<-release
				panic("boom")
			}

			return &testConfig{Addr: "recovered"}
		})

		go func() {
			defer func() { _ = recover() }()
			_, _ = Resolve[*testConfig](c)
		}()
		<-started

		waiter := make(chan error)
		go func() {
			_, err := Resolve[*testConfig](c)
			waiter <- err
		}()

		waitUntil(t, func() bool {
			c.mu.RLock()
			defer c.mu.RUnlock()

			return len(c.waiting) == 1
		})
		close(release)

		select {
		case err := <-waiter:
			if !errors.Is(err, ErrFactoryFailed) || !strings.Contains(err.Error(), "panicked") {
				t.Fatalf("expected panic error, got %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("waiter was not released")
		}

		if got := mustResolve[*testConfig](t, c); got.Addr != "recovered" {
			t.Fatalf("unexpected addr %q", got.Addr)
		}
		assertNoBuildState(t, c)
	})
}

type nodeA struct{}
type nodeB struct{}
type nodeC struct{}

func TestCircularDependency(t *testing.T) {
	t.Run("self", func(t *testing.T) {
		c := NewContainer()
		Register(c, func() (*nodeA, error) {
			_, err := Resolve[*nodeA](c)

			return &nodeA{}, err
		})

		_, err := Resolve[*nodeA](c)
		if !errors.Is(err, ErrCircularDependency) {
			t.Fatalf("expected ErrCircularDependency, got %v", err)
		}
		if !strings.Contains(err.Error(), "circular dependency: *di.nodeA -> *di.nodeA") {
			t.Fatalf("unexpected message %q", err.Error())
		}
		assertNoBuildState(t, c)
	})

	t.Run("two factories", func(t *testing.T) {
		c := NewContainer()
		Register(c, func() (*nodeA, error) {
			_, err := Resolve[*nodeB](c)

			return &nodeA{}, err
		})
		Register(c, func() (*nodeB, error) {
			_, err := Resolve[*nodeA](c)

			return &nodeB{}, err
		})

		_, err := Resolve[*nodeA](c)
		if !errors.Is(err, ErrCircularDependency) {
			t.Fatalf("expected ErrCircularDependency, got %v", err)
		}
		if !strings.Contains(err.Error(), "circular dependency: *di.nodeA -> *di.nodeB -> *di.nodeA") {
			t.Fatalf("unexpected message %q", err.Error())
		}
		assertNoBuildState(t, c)
	})

	t.Run("three factories", func(t *testing.T) {
		c := NewContainer()
		Register(c, func() (*nodeA, error) {
			_, err := Resolve[*nodeB](c)

			return &nodeA{}, err
		})
		Register(c, func() (*nodeB, error) {
			_, err := Resolve[*nodeC](c)

			return &nodeB{}, err
		})
		Register(c, func() (*nodeC, error) {
			_, err := Resolve[*nodeA](c)

			return &nodeC{}, err
		})

		_, err := Resolve[*nodeB](c)
		if !strings.Contains(err.Error(), "circular dependency: *di.nodeB -> *di.nodeC -> *di.nodeA -> *di.nodeB") {
			t.Fatalf("unexpected message %q", err.Error())
		}
	})

	t.Run("factory ignoring the error still completes", func(t *testing.T) {
		c := NewContainer()
		Register(c, func() *nodeA {
			_, _ = Resolve[*nodeB](c)

			return &nodeA{}
		})
		Register(c, func() *nodeB {
			_, _ = Resolve[*nodeA](c)

			return &nodeB{}
		})

		mustResolve[*nodeA](t, c)
		mustResolve[*nodeB](t, c)
		assertNoBuildState(t, c)
	})

	t.Run("container stays usable after cycle", func(t *testing.T) {
		c := NewContainer()
		var cycle atomic.Bool
		cycle.Store(true)
		Register(c, func() (*nodeA, error) {
			if cycle.Load() {
				if _, err := Resolve[*nodeA](c); err != nil {
					return nil, err
				}
			}

			return &nodeA{}, nil
		})

		expectErr[*nodeA](t, c, ErrCircularDependency)
		cycle.Store(false)
		mustResolve[*nodeA](t, c)
	})

	t.Run("across goroutines", func(t *testing.T) {
		c := NewContainer()
		aStarted, bStarted := make(chan struct{}), make(chan struct{})
		Register(c, func() (*nodeA, error) {
			close(aStarted)
			<-bStarted
			_, err := Resolve[*nodeB](c)

			return &nodeA{}, err
		})
		Register(c, func() (*nodeB, error) {
			close(bStarted)
			<-aStarted
			_, err := Resolve[*nodeA](c)

			return &nodeB{}, err
		})

		errs := make(chan error, 2)
		go func() { _, err := Resolve[*nodeA](c); errs <- err }()
		go func() { _, err := Resolve[*nodeB](c); errs <- err }()

		for i := 0; i < 2; i++ {
			select {
			case err := <-errs:
				if !errors.Is(err, ErrCircularDependency) {
					t.Fatalf("expected ErrCircularDependency, got %v", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("deadlock")
			}
		}
		assertNoBuildState(t, c)
	})

	t.Run("diamond is not a cycle", func(t *testing.T) {
		// A -> B, A -> C, B -> C
		c := NewContainer()
		Register(c, func() *nodeC { return &nodeC{} })
		Register(c, func() *nodeB { mustResolve[*nodeC](t, c); return &nodeB{} })
		Register(c, func() *nodeA {
			mustResolve[*nodeB](t, c)
			mustResolve[*nodeC](t, c)

			return &nodeA{}
		})

		mustResolve[*nodeA](t, c)
	})

	t.Run("dependency re-registered while awaited is not a cycle", func(t *testing.T) {
		c := NewContainer()
		bStarted, releaseB := make(chan struct{}), make(chan struct{})
		Register(c, func() *nodeB {
			close(bStarted)
			<-releaseB

			return &nodeB{}
		})
		Register(c, func() *nodeA {
			mustResolve[*nodeB](t, c)

			return &nodeA{}
		})

		done := make(chan struct{}, 3)
		go func() { mustResolve[*nodeB](t, c); done <- struct{}{} }()
		<-bStarted
		go func() { mustResolve[*nodeA](t, c); done <- struct{}{} }()

		// A создаётся и ждёт B
		waitUntil(t, func() bool {
			c.mu.RLock()
			defer c.mu.RUnlock()

			return len(c.waiting) == 1
		})

		// B перерегистрирован: цепочка ожидания обрывается на новой фабрике
		Register(c, func() *nodeB { return &nodeB{} })

		go func() {
			if _, err := Resolve[*nodeA](c); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			done <- struct{}{}
		}()

		waitUntil(t, func() bool {
			c.mu.RLock()
			defer c.mu.RUnlock()

			return len(c.waiting) == 2
		})
		close(releaseB)

		for i := 0; i < 3; i++ {
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("deadlock")
			}
		}
		assertNoBuildState(t, c)
	})

	t.Run("concurrent waiter is not a cycle", func(t *testing.T) {
		c := NewContainer()
		Register(c, func() *nodeA {
			time.Sleep(5 * time.Millisecond)

			return &nodeA{}
		})
		Register(c, func() *nodeB {
			mustResolve[*nodeA](t, c)

			return &nodeB{}
		})

		errs := make(chan error, 2)
		go func() { _, err := Resolve[*nodeA](c); errs <- err }()
		go func() { _, err := Resolve[*nodeB](c); errs <- err }()

		for i := 0; i < 2; i++ {
			if err := <-errs; err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		}
	})
}

func TestGoroutineID(t *testing.T) {
	main := goroutineID()
	if main <= 0 {
		t.Fatalf("invalid id %d", main)
	}
	if goroutineID() != main {
		t.Fatal("id must be stable within goroutine")
	}

	other := make(chan int64)
	go func() { other <- goroutineID() }()

	if <-other == main {
		t.Fatal("different goroutines must have different ids")
	}
}

// resolveConcurrently - одновременно резолвит T из n горутин и возвращает ошибки
func resolveConcurrently[T any](c *Container, n int) []error {
	errs := make([]error, n)
	start := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			_, errs[i] = Resolve[T](c)
		}(i)
	}
	close(start)
	wg.Wait()

	return errs
}

// waitUntil - ждёт выполнения условия
func waitUntil(t *testing.T, cond func() bool) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition not met")
		}
		time.Sleep(time.Millisecond)
	}
}

// assertNoBuildState - после всех резолвов не должно остаться служебного состояния
func assertNoBuildState(t *testing.T, c *Container) {
	t.Helper()

	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.stacks) != 0 || len(c.waiting) != 0 {
		t.Fatalf("leaked build state: stacks=%v waiting=%v", c.stacks, c.waiting)
	}
	for typ, f := range c.functions {
		if f.building != nil {
			t.Fatalf("factory %s left in building state", typ)
		}
	}
}

func TestParseGoroutineID(t *testing.T) {
	cases := map[string]int64{
		"goroutine 42 [running]:\nmain.main()": 42,
		"goroutine 1 [running]:":               1,
		"goroutine 9223372036854775807 [x]":    9223372036854775807,
		"":                                     unknownGoroutine,
		"goroutine ":                           unknownGoroutine,
		"goroutine 42":                         unknownGoroutine,
		"goroutine abc [running]:":             unknownGoroutine,
		"goroutine -5 [running]:":              unknownGoroutine,
		"goroutine 0 [running]:":               unknownGoroutine,
		"goroutine 99999999999999999999 [x]":   unknownGoroutine,
		"thread 42 [running]:":                 unknownGoroutine,
		"Goroutine 42 [running]:":              unknownGoroutine,
	}

	for input, want := range cases {
		if got := parseGoroutineID([]byte(input)); got != want {
			t.Errorf("parseGoroutineID(%q) = %d, want %d", input, got, want)
		}
	}
}

// withoutGoroutineID - имитирует Go, в котором формат заголовка стека не распознаётся
func withoutGoroutineID(t *testing.T) {
	original := goroutineID
	goroutineID = func() int64 { return unknownGoroutine }
	t.Cleanup(func() { goroutineID = original })
}

func TestWithoutGoroutineID(t *testing.T) {
	t.Run("factory singleton", func(t *testing.T) {
		withoutGoroutineID(t)

		c := NewContainer()
		var calls atomic.Int32
		Register(c, func() service {
			calls.Add(1)
			time.Sleep(time.Millisecond)

			return &serviceImpl{name: "svc"}
		})

		for i, err := range resolveConcurrently[service](c, 50) {
			if err != nil {
				t.Fatalf("goroutine %d: %v", i, err)
			}
		}
		if got := calls.Load(); got != 1 {
			t.Fatalf("factory called %d times, want 1", got)
		}
		assertNoBuildState(t, c)
	})

	t.Run("shared error", func(t *testing.T) {
		withoutGoroutineID(t)

		c := NewContainer()
		cause := errors.New("db down")
		Register(c, func() (*testConfig, error) {
			time.Sleep(time.Millisecond)

			return nil, cause
		})

		for i, err := range resolveConcurrently[*testConfig](c, 50) {
			if !errors.Is(err, cause) {
				t.Fatalf("goroutine %d: expected cause, got %v", i, err)
			}
		}
		assertNoBuildState(t, c)
	})

	t.Run("nested factories", func(t *testing.T) {
		withoutGoroutineID(t)

		c := NewContainer()
		Register(c, func() *nodeC { return &nodeC{} })
		Register(c, func() *nodeB { mustResolve[*nodeC](t, c); return &nodeB{} })
		Register(c, func() *nodeA { mustResolve[*nodeB](t, c); return &nodeA{} })

		mustResolve[*nodeA](t, c)
		assertNoBuildState(t, c)
	})

	t.Run("concurrent waiter is not a false cycle", func(t *testing.T) {
		withoutGoroutineID(t)

		c := NewContainer()
		Register(c, func() *nodeA {
			time.Sleep(5 * time.Millisecond)

			return &nodeA{}
		})
		Register(c, func() *nodeB {
			mustResolve[*nodeA](t, c)

			return &nodeB{}
		})

		errs := make(chan error, 2)
		go func() { _, err := Resolve[*nodeA](c); errs <- err }()
		go func() { _, err := Resolve[*nodeB](c); errs <- err }()

		for i := 0; i < 2; i++ {
			if err := <-errs; err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		}
		assertNoBuildState(t, c)
	})

	t.Run("mixed with known ids", func(t *testing.T) {
		c := NewContainer()
		started, release := make(chan struct{}), make(chan struct{})
		Register(c, func() *nodeA {
			close(started)
			<-release

			return &nodeA{}
		})

		// Фабрику вызывает горутина без id
		original := goroutineID
		goroutineID = func() int64 { return unknownGoroutine }
		result := make(chan error)
		go func() { _, err := Resolve[*nodeA](c); result <- err }()
		<-started
		goroutineID = original

		// Горутина с id ждёт её: цепочка обрывается, ложного цикла нет
		waiter := make(chan error)
		go func() { _, err := Resolve[*nodeA](c); waiter <- err }()
		waitUntil(t, func() bool {
			c.mu.RLock()
			defer c.mu.RUnlock()

			return len(c.waiting) == 1
		})
		close(release)

		if err := <-result; err != nil {
			t.Fatalf("builder: %v", err)
		}
		if err := <-waiter; err != nil {
			t.Fatalf("waiter: %v", err)
		}
		assertNoBuildState(t, c)
	})
}
