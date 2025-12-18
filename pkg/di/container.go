package di

import (
	"errors"
	"reflect"
	"sync"
)

// NewContainer - конструктор контейнера
func NewContainer() *Container {
	return &Container{
		instances: make(map[reflect.Type]interface{}),
		functions: make(map[reflect.Type]interface{}),
	}
}

// Container - контейнер зависимостей с поддержкой фабрик
type Container struct {
	mu        sync.RWMutex
	instances map[reflect.Type]interface{}
	functions map[reflect.Type]interface{}
}

// Register - регистрирует зависимость (структуру или указатель)
func Register[T any](c *Container, instance T) {
	typ := reflect.TypeOf(instance)

	// Если это функция, регистрируем как конструктор
	if typ.Kind() == reflect.Func {
		outType := typ.Out(0) // Первый возвращаемый тип
		c.mu.Lock()
		c.functions[outType] = instance
		c.mu.Unlock()

		return
	}

	c.mu.Lock()
	c.instances[typ] = instance
	c.mu.Unlock()
}

// Resolve - получает зависимость, используя дженерики (Go 1.18+)
func Resolve[T any](c *Container) (T, error) {
	var zero T
	typ := reflect.TypeOf((*T)(nil)).Elem() // Универсальный тип
	c.mu.RLock()

	// Проверяем, есть ли готовый объект
	if instance, exists := c.instances[typ]; exists {
		c.mu.RUnlock()

		return instance.(T), nil
	}

	// Проверяем, есть ли функции
	function, exists := c.functions[typ]
	c.mu.RUnlock()

	if !exists {
		return zero, errors.New("dependency not found: " + typ.String())
	}

	// Если это фабрика (функция), вызываем её
	if factory, ok := function.(func() T); ok {
		instance := factory()
		Register(c, instance)
		c.mu.Lock()
		delete(c.functions, typ)
		c.mu.Unlock()

		return instance, nil
	}

	return zero, errors.New("wrong dependency type: " + typ.String())
}
