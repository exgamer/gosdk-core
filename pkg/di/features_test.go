package di

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
)

type replicaDB struct{ dsn string }

func TestNamed(t *testing.T) {
	t.Run("several instances of one type", func(t *testing.T) {
		c := NewContainer()
		RegisterNamed(c, "primary", &replicaDB{dsn: "primary"})
		RegisterNamed(c, "replica", &replicaDB{dsn: "replica"})

		if got := MustResolveNamed[*replicaDB](c, "primary"); got.dsn != "primary" {
			t.Fatalf("unexpected dsn %q", got.dsn)
		}
		if got := MustResolveNamed[*replicaDB](c, "replica"); got.dsn != "replica" {
			t.Fatalf("unexpected dsn %q", got.dsn)
		}
	})

	t.Run("named and unnamed are separate", func(t *testing.T) {
		c := NewContainer()
		Register(c, &replicaDB{dsn: "default"})
		RegisterNamed(c, "replica", &replicaDB{dsn: "replica"})

		if MustResolve[*replicaDB](c).dsn != "default" {
			t.Fatal("unnamed")
		}
		if MustResolveNamed[*replicaDB](c, "replica").dsn != "replica" {
			t.Fatal("named")
		}
	})

	t.Run("empty name is unnamed", func(t *testing.T) {
		c := NewContainer()
		RegisterNamed(c, "", &replicaDB{dsn: "default"})

		if MustResolve[*replicaDB](c).dsn != "default" {
			t.Fatal("expected unnamed")
		}
	})

	t.Run("named only, unnamed not found", func(t *testing.T) {
		c := NewContainer()
		RegisterNamed(c, "replica", &replicaDB{})

		expectErr[*replicaDB](t, c, ErrDependencyNotFound)
	})

	t.Run("not found message contains name", func(t *testing.T) {
		_, err := ResolveNamed[*replicaDB](NewContainer(), "replica")

		if !errors.Is(err, ErrDependencyNotFound) {
			t.Fatalf("expected ErrDependencyNotFound, got %v", err)
		}
		if err.Error() != `dependency not found: *di.replicaDB named "replica"` {
			t.Fatalf("unexpected message %q", err.Error())
		}
	})

	t.Run("named factories are separate singletons", func(t *testing.T) {
		c := NewContainer()
		RegisterNamed(c, "primary", func() *replicaDB { return &replicaDB{dsn: "primary"} })
		RegisterNamed(c, "replica", func() (*replicaDB, error) { return &replicaDB{dsn: "replica"}, nil })

		primary := MustResolveNamed[*replicaDB](c, "primary")
		replica := MustResolveNamed[*replicaDB](c, "replica")

		if primary == replica || primary.dsn != "primary" || replica.dsn != "replica" {
			t.Fatal("expected two different singletons")
		}
		if MustResolveNamed[*replicaDB](c, "primary") != primary {
			t.Fatal("expected cached primary")
		}
	})

	t.Run("named interface", func(t *testing.T) {
		c := NewContainer()
		RegisterNamed[service](c, "a", &serviceImpl{name: "a"})
		RegisterNamed[service](c, "b", &serviceImpl{name: "b"})

		if MustResolveNamed[service](c, "a").Name() != "a" || MustResolveNamed[service](c, "b").Name() != "b" {
			t.Fatal("named interfaces")
		}
	})

	t.Run("last registration wins per name", func(t *testing.T) {
		c := NewContainer()
		RegisterNamed(c, "replica", &replicaDB{dsn: "first"})
		RegisterNamed(c, "replica", func() *replicaDB { return &replicaDB{dsn: "second"} })

		if MustResolveNamed[*replicaDB](c, "replica").dsn != "second" {
			t.Fatal("expected last registration")
		}
	})

	t.Run("factory error message contains name", func(t *testing.T) {
		c := NewContainer()
		RegisterNamed(c, "replica", func() (*replicaDB, error) { return nil, errors.New("down") })

		_, err := ResolveNamed[*replicaDB](c, "replica")
		if err.Error() != `factory failed: *di.replicaDB named "replica": down` {
			t.Fatalf("unexpected message %q", err.Error())
		}
	})

	t.Run("cycle path contains names", func(t *testing.T) {
		c := NewContainer()
		RegisterNamed(c, "a", func() (*replicaDB, error) {
			_, err := ResolveNamed[*replicaDB](c, "b")

			return &replicaDB{}, err
		})
		RegisterNamed(c, "b", func() (*replicaDB, error) {
			_, err := ResolveNamed[*replicaDB](c, "a")

			return &replicaDB{}, err
		})

		_, err := ResolveNamed[*replicaDB](c, "a")
		want := `circular dependency: *di.replicaDB named "a" -> *di.replicaDB named "b" -> *di.replicaDB named "a"`
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("unexpected message %q", err.Error())
		}
	})

	t.Run("named depends on another name of same type is not a cycle", func(t *testing.T) {
		c := NewContainer()
		RegisterNamed(c, "base", func() *replicaDB { return &replicaDB{dsn: "base"} })
		RegisterNamed(c, "derived", func() *replicaDB {
			return &replicaDB{dsn: MustResolveNamed[*replicaDB](c, "base").dsn + "+derived"}
		})

		if got := MustResolveNamed[*replicaDB](c, "derived"); got.dsn != "base+derived" {
			t.Fatalf("unexpected dsn %q", got.dsn)
		}
	})
}

func TestMustResolve(t *testing.T) {
	t.Run("returns instance", func(t *testing.T) {
		c := NewContainer()
		cfg := &testConfig{}
		Register(c, cfg)

		if MustResolve[*testConfig](c) != cfg {
			t.Fatal("expected same instance")
		}
	})

	t.Run("panics with not found error", func(t *testing.T) {
		err := recoverError(func() { MustResolve[*testConfig](NewContainer()) })

		if !errors.Is(err, ErrDependencyNotFound) {
			t.Fatalf("expected ErrDependencyNotFound panic, got %v", err)
		}
	})

	t.Run("panics with factory error", func(t *testing.T) {
		c := NewContainer()
		cause := errors.New("down")
		Register(c, func() (*testConfig, error) { return nil, cause })

		err := recoverError(func() { MustResolve[*testConfig](c) })

		if !errors.Is(err, ErrFactoryFailed) || !errors.Is(err, cause) {
			t.Fatalf("expected factory error panic, got %v", err)
		}
	})

	t.Run("named panics with name in error", func(t *testing.T) {
		err := recoverError(func() { MustResolveNamed[*testConfig](NewContainer(), "x") })

		if err == nil || !strings.Contains(err.Error(), `named "x"`) {
			t.Fatalf("unexpected panic %v", err)
		}
	})
}

func TestHas(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		if Has[*testConfig](NewContainer()) {
			t.Fatal("expected false")
		}
	})

	t.Run("instance", func(t *testing.T) {
		c := NewContainer()
		Register(c, &testConfig{})

		if !Has[*testConfig](c) {
			t.Fatal("expected true")
		}
		if Has[testConfig](c) {
			t.Fatal("value type must not match pointer")
		}
	})

	t.Run("factory is not called", func(t *testing.T) {
		c := NewContainer()
		var calls int
		Register(c, func() *testConfig {
			calls++

			return &testConfig{}
		})

		if !Has[*testConfig](c) {
			t.Fatal("expected true")
		}
		if calls != 0 {
			t.Fatal("factory must not be called")
		}
	})

	t.Run("after factory resolved", func(t *testing.T) {
		c := NewContainer()
		Register(c, func() *testConfig { return &testConfig{} })
		MustResolve[*testConfig](c)

		if !Has[*testConfig](c) {
			t.Fatal("expected true")
		}
	})

	t.Run("failing factory still registered", func(t *testing.T) {
		c := NewContainer()
		Register(c, func() (*testConfig, error) { return nil, errors.New("down") })
		_, _ = Resolve[*testConfig](c)

		if !Has[*testConfig](c) {
			t.Fatal("expected true")
		}
	})

	t.Run("interface by static type", func(t *testing.T) {
		c := NewContainer()
		Register[service](c, &serviceImpl{})

		if !Has[service](c) || Has[*serviceImpl](c) {
			t.Fatal("expected only interface")
		}
	})

	t.Run("named", func(t *testing.T) {
		c := NewContainer()
		RegisterNamed(c, "replica", &replicaDB{})

		if !HasNamed[*replicaDB](c, "replica") {
			t.Fatal("expected named")
		}
		if Has[*replicaDB](c) || HasNamed[*replicaDB](c, "primary") {
			t.Fatal("other names must be absent")
		}
	})
}

// closer - фиксирует порядок закрытия
type closer struct {
	name string
	log  *closeLog
	err  error
}

func (c *closer) Close() error {
	c.log.add(c.name)

	return c.err
}

type closeLog struct {
	mu    sync.Mutex
	names []string
}

func (l *closeLog) add(name string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.names = append(l.names, name)
}

type silentCloser struct{ closed bool }

func (c *silentCloser) Close() { c.closed = true }

type shutdowner struct {
	ctx    context.Context
	closed bool
}

func (s *shutdowner) Shutdown(ctx context.Context) error {
	s.ctx = ctx

	return nil
}

func (s *shutdowner) Close() error {
	s.closed = true

	return nil
}

type panicCloser struct{}

func (panicCloser) Close() error { panic("close boom") }

func TestClose(t *testing.T) {
	t.Run("reverse creation order", func(t *testing.T) {
		c := NewContainer()
		log := &closeLog{}
		RegisterNamed(c, "db", func() *closer { return &closer{name: "db", log: log} })
		RegisterNamed(c, "repo", func() *closer {
			MustResolveNamed[*closer](c, "db")

			return &closer{name: "repo", log: log}
		})
		RegisterNamed(c, "service", func() *closer {
			MustResolveNamed[*closer](c, "repo")

			return &closer{name: "service", log: log}
		})

		MustResolveNamed[*closer](c, "service")

		if err := c.Close(context.Background()); err != nil {
			t.Fatalf("close: %v", err)
		}
		if want := []string{"service", "repo", "db"}; !slices.Equal(log.names, want) {
			t.Fatalf("close order %v, want %v", log.names, want)
		}
	})

	t.Run("registered instances are not closed", func(t *testing.T) {
		c := NewContainer()
		log := &closeLog{}
		Register(c, &closer{name: "external", log: log})

		if err := c.Close(context.Background()); err != nil {
			t.Fatalf("close: %v", err)
		}
		if len(log.names) != 0 {
			t.Fatalf("external instance closed: %v", log.names)
		}
	})

	t.Run("unresolved factories are not called", func(t *testing.T) {
		c := NewContainer()
		var calls int
		Register(c, func() *closer {
			calls++

			return &closer{log: &closeLog{}}
		})

		if err := c.Close(context.Background()); err != nil {
			t.Fatalf("close: %v", err)
		}
		if calls != 0 {
			t.Fatal("factory must not be called on close")
		}
	})

	t.Run("failed factory result is not closed", func(t *testing.T) {
		c := NewContainer()
		log := &closeLog{}
		Register(c, func() (*closer, error) { return &closer{name: "failed", log: log}, errors.New("down") })
		_, _ = Resolve[*closer](c)

		if err := c.Close(context.Background()); err != nil {
			t.Fatalf("close: %v", err)
		}
		if len(log.names) != 0 {
			t.Fatalf("failed result closed: %v", log.names)
		}
	})

	t.Run("instance replaced by re-registration is still closed", func(t *testing.T) {
		c := NewContainer()
		log := &closeLog{}
		Register(c, func() *closer { return &closer{name: "old", log: log} })
		MustResolve[*closer](c)
		Register(c, func() *closer { return &closer{name: "new", log: log} })
		MustResolve[*closer](c)

		if err := c.Close(context.Background()); err != nil {
			t.Fatalf("close: %v", err)
		}
		if want := []string{"new", "old"}; !slices.Equal(log.names, want) {
			t.Fatalf("closed %v, want %v", log.names, want)
		}
	})

	t.Run("errors are joined and do not stop closing", func(t *testing.T) {
		c := NewContainer()
		log := &closeLog{}
		errA, errB := errors.New("a failed"), errors.New("b failed")
		RegisterNamed(c, "a", func() *closer { return &closer{name: "a", log: log, err: errA} })
		RegisterNamed(c, "b", func() *closer { return &closer{name: "b", log: log, err: errB} })
		RegisterNamed(c, "c", func() *closer { return &closer{name: "c", log: log} })
		MustResolveNamed[*closer](c, "a")
		MustResolveNamed[*closer](c, "b")
		MustResolveNamed[*closer](c, "c")

		err := c.Close(context.Background())

		if !errors.Is(err, errA) || !errors.Is(err, errB) {
			t.Fatalf("expected both errors, got %v", err)
		}
		if !strings.Contains(err.Error(), `close *di.closer named "a": a failed`) {
			t.Fatalf("unexpected message %q", err.Error())
		}
		if len(log.names) != 3 {
			t.Fatalf("expected all closed, got %v", log.names)
		}
	})

	t.Run("close without error", func(t *testing.T) {
		c := NewContainer()
		Register(c, func() *silentCloser { return &silentCloser{} })
		s := MustResolve[*silentCloser](c)

		if err := c.Close(context.Background()); err != nil {
			t.Fatalf("close: %v", err)
		}
		if !s.closed {
			t.Fatal("expected closed")
		}
	})

	t.Run("shutdown preferred over close and gets context", func(t *testing.T) {
		c := NewContainer()
		Register(c, func() *shutdowner { return &shutdowner{} })
		s := MustResolve[*shutdowner](c)

		type ctxKey struct{}
		ctx := context.WithValue(context.Background(), ctxKey{}, "v")

		if err := c.Close(ctx); err != nil {
			t.Fatalf("close: %v", err)
		}
		if s.ctx != ctx {
			t.Fatal("expected shutdown with given context")
		}
		if s.closed {
			t.Fatal("close must not be called when shutdown exists")
		}
	})

	t.Run("panic in close becomes error", func(t *testing.T) {
		c := NewContainer()
		log := &closeLog{}
		RegisterNamed(c, "ok", func() *closer { return &closer{name: "ok", log: log} })
		Register(c, func() panicCloser { return panicCloser{} })
		MustResolveNamed[*closer](c, "ok")
		MustResolve[panicCloser](c)

		err := c.Close(context.Background())

		if err == nil || !strings.Contains(err.Error(), "panic: close boom") {
			t.Fatalf("expected panic error, got %v", err)
		}
		if len(log.names) != 1 {
			t.Fatal("other instances must be closed")
		}
	})

	t.Run("nil and non-closers are skipped", func(t *testing.T) {
		c := NewContainer()
		Register(c, func() *closer { return nil })
		Register(c, func() service { return nil })
		Register(c, func() *testConfig { return &testConfig{} })
		Register(c, func() int { return 1 })
		MustResolve[*closer](c)
		MustResolve[service](c)
		MustResolve[*testConfig](c)
		MustResolve[int](c)

		if err := c.Close(context.Background()); err != nil {
			t.Fatalf("close: %v", err)
		}
	})

	t.Run("second close does nothing", func(t *testing.T) {
		c := NewContainer()
		log := &closeLog{}
		Register(c, func() *closer { return &closer{name: "once", log: log} })
		MustResolve[*closer](c)

		_ = c.Close(context.Background())
		_ = c.Close(context.Background())

		if len(log.names) != 1 {
			t.Fatalf("closed %d times, want 1", len(log.names))
		}
	})

	t.Run("empty container", func(t *testing.T) {
		if err := NewContainer().Close(context.Background()); err != nil {
			t.Fatalf("close: %v", err)
		}
	})

	t.Run("concurrent close closes once", func(t *testing.T) {
		c := NewContainer()
		log := &closeLog{}
		Register(c, func() *closer { return &closer{name: "once", log: log} })
		MustResolve[*closer](c)

		var wg sync.WaitGroup
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = c.Close(context.Background())
			}()
		}
		wg.Wait()

		if len(log.names) != 1 {
			t.Fatalf("closed %d times, want 1", len(log.names))
		}
	})
}

func TestIsNil(t *testing.T) {
	var (
		nilPtr   *testConfig
		nilMap   map[string]int
		nilSlice []int
		nilFunc  func()
		nilChan  chan int
	)

	for name, v := range map[string]interface{}{
		"nil": nil, "ptr": nilPtr, "map": nilMap, "slice": nilSlice, "func": nilFunc, "chan": nilChan,
	} {
		if !isNil(v) {
			t.Errorf("%s: expected nil", name)
		}
	}

	for name, v := range map[string]interface{}{
		"int": 0, "struct": testConfig{}, "ptr": &testConfig{}, "string": "",
	} {
		if isNil(v) {
			t.Errorf("%s: expected not nil", name)
		}
	}
}

// recoverError - вызывает fn и возвращает ошибку, с которой она запаниковала
func recoverError(fn func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err, _ = r.(error)
		}
	}()

	fn()

	return nil
}
