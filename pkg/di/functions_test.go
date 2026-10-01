package di

import (
	"errors"
	"testing"
	"time"

	"github.com/exgamer/gosdk-core/pkg/config"
)

func TestGetLocation(t *testing.T) {
	t.Run("registered", func(t *testing.T) {
		c := NewContainer()
		loc := time.FixedZone("UTC+5", 5*60*60)
		Register(c, loc)

		got, err := GetLocation(c)
		if err != nil {
			t.Fatalf("get location: %v", err)
		}
		if got != loc {
			t.Fatal("expected same location")
		}
	})

	t.Run("not registered", func(t *testing.T) {
		got, err := GetLocation(NewContainer())
		if !errors.Is(err, ErrDependencyNotFound) {
			t.Fatalf("expected ErrDependencyNotFound, got %v", err)
		}
		if got != nil {
			t.Fatal("expected nil")
		}
	})
}

func TestGetBaseConfig(t *testing.T) {
	t.Run("registered", func(t *testing.T) {
		c := NewContainer()
		cfg := &config.BaseConfig{Name: "app"}
		Register(c, cfg)

		got, err := GetBaseConfig(c)
		if err != nil {
			t.Fatalf("get base config: %v", err)
		}
		if got != cfg {
			t.Fatal("expected same config")
		}
	})

	t.Run("registered via factory", func(t *testing.T) {
		c := NewContainer()
		Register(c, func() *config.BaseConfig { return &config.BaseConfig{Name: "lazy"} })

		got, err := GetBaseConfig(c)
		if err != nil {
			t.Fatalf("get base config: %v", err)
		}
		if got.Name != "lazy" {
			t.Fatalf("unexpected name %q", got.Name)
		}
	})

	t.Run("value registered, pointer not found", func(t *testing.T) {
		c := NewContainer()
		Register(c, config.BaseConfig{Name: "app"})

		if _, err := GetBaseConfig(c); !errors.Is(err, ErrDependencyNotFound) {
			t.Fatalf("expected ErrDependencyNotFound, got %v", err)
		}
	})

	t.Run("not registered", func(t *testing.T) {
		got, err := GetBaseConfig(NewContainer())
		if !errors.Is(err, ErrDependencyNotFound) {
			t.Fatalf("expected ErrDependencyNotFound, got %v", err)
		}
		if got != nil {
			t.Fatal("expected nil")
		}
	})
}
