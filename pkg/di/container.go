package di

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
)

var (
	// ErrDependencyNotFound - зависимость не зарегистрирована
	ErrDependencyNotFound = errors.New("dependency not found")
	// ErrFactoryFailed - фабрика вернула ошибку или запаниковала
	ErrFactoryFailed = errors.New("factory failed")
	// ErrCircularDependency - фабрики зависят друг от друга по кругу
	ErrCircularDependency = errors.New("circular dependency")
)

var errorType = reflect.TypeFor[error]()

// key - ключ зависимости: тип и необязательное имя
type key struct {
	typ  reflect.Type
	name string
}

func (k key) String() string {
	if k.name == "" {
		return k.typ.String()
	}

	return fmt.Sprintf("%s named %q", k.typ, k.name)
}

// factory - зарегистрированная фабрика, приведённая к единому виду
type factory struct {
	create   func() (interface{}, error)
	building *attempt // текущая попытка создания, nil - никто не создаёт
}

// attempt - попытка создания зависимости, которую ждут остальные горутины
type attempt struct {
	owner int64         // горутина, вызвавшая фабрику
	done  chan struct{} // закрывается по завершении попытки
	err   error         // ошибка попытки, общая для всех ожидающих
}

// created - объект, созданный фабрикой контейнера
type created struct {
	key      key
	instance interface{}
}

// NewContainer - конструктор контейнера
func NewContainer() *Container {
	return &Container{
		instances: make(map[key]interface{}),
		functions: make(map[key]*factory),
		stacks:    make(map[int64][]key),
		waiting:   make(map[int64]key),
	}
}

// Container - контейнер зависимостей с поддержкой фабрик
type Container struct {
	mu        sync.RWMutex
	instances map[key]interface{}
	functions map[key]*factory
	created   []created       // созданное фабриками, в порядке создания - для Close
	stacks    map[int64][]key // что сейчас создаёт каждая горутина, по порядку вложенности
	waiting   map[int64]key   // чьё создание ждёт каждая горутина
}

// Register - регистрирует зависимость (структуру, указатель, интерфейс или фабрику).
// Ключом служит статический тип T, а не динамический тип значения,
// поэтому интерфейс резолвится по интерфейсу: Register[Svc](c, impl) -> Resolve[Svc].
//
// Функция регистрируется как фабрика своего возвращаемого типа.
// Поддерживаются func() T и func() (T, error), на остальные сигнатуры - паника.
//
// Побеждает последняя регистрация: новая фабрика заменяет уже созданный инстанс,
// новый инстанс заменяет фабрику.
func Register[T any](c *Container, instance T) {
	RegisterNamed(c, "", instance)
}

// RegisterNamed - как Register, но под именем: несколько зависимостей одного типа.
// Пустое имя - то же, что Register.
func RegisterNamed[T any](c *Container, name string, instance T) {
	typ := reflect.TypeFor[T]()

	if typ.Kind() == reflect.Func {
		f := newFactory(reflect.ValueOf(instance), typ)
		k := key{typ: typ.Out(0), name: name}

		c.mu.Lock()
		c.functions[k] = f
		delete(c.instances, k)
		c.mu.Unlock()

		return
	}

	k := key{typ: typ, name: name}

	c.mu.Lock()
	c.instances[k] = instance
	delete(c.functions, k)
	c.mu.Unlock()
}

// Resolve - получает зависимость. Фабрика вызывается при первом запросе ровно один раз,
// результат кешируется как синглтон под запрошенным типом T.
// Конкурентные запросы ждут результата этого вызова.
// Если фабрика вернула ошибку, результат не кешируется и следующий Resolve повторит вызов.
func Resolve[T any](c *Container) (T, error) {
	return ResolveNamed[T](c, "")
}

// ResolveNamed - как Resolve, но для зависимости, зарегистрированной через RegisterNamed
func ResolveNamed[T any](c *Container, name string) (T, error) {
	instance, err := c.resolve(key{typ: reflect.TypeFor[T](), name: name})
	if err != nil {
		var zero T

		return zero, err
	}

	return cast[T](instance), nil
}

// MustResolve - как Resolve, но паникует ошибкой вместо её возврата.
// Для старта приложения, где без зависимости продолжать нельзя.
func MustResolve[T any](c *Container) T {
	return MustResolveNamed[T](c, "")
}

// MustResolveNamed - как ResolveNamed, но паникует ошибкой вместо её возврата
func MustResolveNamed[T any](c *Container, name string) T {
	instance, err := ResolveNamed[T](c, name)
	if err != nil {
		panic(err)
	}

	return instance
}

// Has - зарегистрирована ли зависимость (инстанс или фабрика). Фабрику не вызывает.
func Has[T any](c *Container) bool {
	return HasNamed[T](c, "")
}

// HasNamed - как Has, но для зависимости, зарегистрированной через RegisterNamed
func HasNamed[T any](c *Container, name string) bool {
	k := key{typ: reflect.TypeFor[T](), name: name}

	c.mu.RLock()
	defer c.mu.RUnlock()

	_, hasInstance := c.instances[k]
	_, hasFactory := c.functions[k]

	return hasInstance || hasFactory
}

// Close - закрывает объекты, созданные фабриками контейнера, в обратном порядке создания:
// зависимый объект закрывается раньше своих зависимостей.
// Готовые инстансы, переданные в Register, не закрываются - ими владеет тот, кто их создал.
//
// Поддерживаются Shutdown(ctx) error, Close() error и Close(). Ошибки собираются через errors.Join,
// ошибка одного объекта не мешает закрыть остальные. Повторный вызов ничего не делает.
func (c *Container) Close(ctx context.Context) error {
	c.mu.Lock()
	toClose := c.created
	c.created = nil
	c.mu.Unlock()

	var errs []error

	for _, item := range slices.Backward(toClose) {
		if err := closeInstance(ctx, item.instance); err != nil {
			errs = append(errs, fmt.Errorf("close %s: %w", item.key, err))
		}
	}

	return errors.Join(errs...)
}

func (c *Container) resolve(k key) (interface{}, error) {
	// Быстрый путь: инстанс уже создан
	c.mu.RLock()
	instance, exists := c.instances[k]
	c.mu.RUnlock()

	if exists {
		return instance, nil
	}

	gid := goroutineID()

	c.mu.Lock()

	for {
		if instance, exists := c.instances[k]; exists {
			c.mu.Unlock()

			return instance, nil
		}

		f, exists := c.functions[k]
		if !exists {
			c.mu.Unlock()

			return nil, fmt.Errorf("%w: %s", ErrDependencyNotFound, k)
		}

		if f.building == nil {
			return c.build(k, f, gid)
		}

		// Фабрику уже кто-то вызывает. Если это мы сами (или ждущая нас горутина) - это цикл
		if path := c.findCycle(k, gid); path != nil {
			c.mu.Unlock()

			return nil, fmt.Errorf("%w: %s", ErrCircularDependency, formatPath(path))
		}

		a := f.building
		if gid != unknownGoroutine {
			c.waiting[gid] = k
		}
		c.mu.Unlock()

		<-a.done

		c.mu.Lock()
		delete(c.waiting, gid)

		if a.err != nil {
			c.mu.Unlock()

			return nil, a.err
		}
	}
}

// build - вызывает фабрику. Вызывается под локом, лок отпускает.
func (c *Container) build(k key, f *factory, gid int64) (interface{}, error) {
	a := &attempt{owner: gid, done: make(chan struct{})}
	f.building = a
	if gid != unknownGoroutine {
		c.stacks[gid] = append(c.stacks[gid], k)
	}
	c.mu.Unlock()

	// При панике фабрики освобождаем ожидающих, паника идёт дальше
	completed := false
	defer func() {
		if !completed {
			c.mu.Lock()
			c.finish(f, a, gid, fmt.Errorf("%w: %s: factory panicked", ErrFactoryFailed, k))
			c.mu.Unlock()
		}
	}()

	// Фабрика вызывается без лока: внутри она может резолвить другие зависимости
	instance, err := f.create()
	completed = true

	c.mu.Lock()
	defer c.mu.Unlock()

	if err != nil {
		err = fmt.Errorf("%w: %s: %w", ErrFactoryFailed, k, err)
		c.finish(f, a, gid, err)

		return nil, err
	}

	c.finish(f, a, gid, nil)

	// Созданное фабрикой закрывает контейнер, даже если его вытеснила перерегистрация
	c.created = append(c.created, created{key: k, instance: instance})

	// Пока работала фабрика, зависимость могли перерегистрировать
	if existing, exists := c.instances[k]; exists {
		return existing, nil
	}

	if c.functions[k] == f {
		c.instances[k] = instance
		delete(c.functions, k)
	}

	return instance, nil
}

// finish - завершает попытку создания и будит ожидающих. Вызывается под локом.
func (c *Container) finish(f *factory, a *attempt, gid int64, err error) {
	if gid != unknownGoroutine {
		if stack := c.stacks[gid][:len(c.stacks[gid])-1]; len(stack) > 0 {
			c.stacks[gid] = stack
		} else {
			delete(c.stacks, gid)
		}
	}

	a.err = err
	f.building = nil
	close(a.done)
}

// findCycle - идёт по цепочке "ключ создаёт горутина X, X ждёт ключ Y, Y создаёт горутина Z...".
// Если цепочка возвращается к gid - это цикл, возвращается его путь. Вызывается под локом.
// Без id горутины поиск отключён: цикл не отличить от обычного ожидания.
func (c *Container) findCycle(start key, gid int64) []key {
	if gid == unknownGoroutine {
		return nil
	}

	var path []key

	for k, steps := start, 0; steps <= len(c.stacks); steps++ {
		f, exists := c.functions[k]
		if !exists || f.building == nil {
			return nil
		}

		owner := f.building.owner
		if owner == unknownGoroutine {
			return nil
		}

		// Владелец положил k в свой стек под тем же локом, что и building, поэтому k там есть
		stack := c.stacks[owner]
		path = append(path, stack[slices.Index(stack, k):]...)

		if owner == gid {
			return append(path, start)
		}

		if k, exists = c.waiting[owner]; !exists {
			return nil
		}
	}

	return nil
}

// newFactory - проверяет сигнатуру и приводит func() T / func() (T, error) к единому виду
func newFactory(fn reflect.Value, typ reflect.Type) *factory {
	withError := typ.NumOut() == 2 && typ.Out(1) == errorType

	if typ.NumIn() != 0 || (typ.NumOut() != 1 && !withError) {
		panic("di: invalid factory " + typ.String() + ": want func() T or func() (T, error)")
	}

	if fn.IsNil() {
		panic("di: nil factory " + typ.String())
	}

	return &factory{
		create: func() (interface{}, error) {
			out := fn.Call(nil)

			if withError && !out[1].IsNil() {
				return nil, out[1].Interface().(error)
			}

			return out[0].Interface(), nil
		},
	}
}

// closeInstance - закрывает объект, если он это умеет. Паника превращается в ошибку,
// чтобы один объект не сорвал закрытие остальных.
func closeInstance(ctx context.Context, instance interface{}) (err error) {
	if isNil(instance) {
		return nil
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	switch v := instance.(type) {
	case interface{ Shutdown(context.Context) error }:
		return v.Shutdown(ctx)
	case io.Closer:
		return v.Close()
	case interface{ Close() }:
		v.Close()
	}

	return nil
}

// isNil - nil-интерфейс или nil-значение ссылочного типа
func isNil(v interface{}) bool {
	if v == nil {
		return true
	}

	switch rv := reflect.ValueOf(v); rv.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan, reflect.Interface:
		return rv.IsNil()
	default:
		return false
	}
}

// unknownGoroutine - id горутины не удалось определить
const unknownGoroutine int64 = 0

// goroutineID - id текущей горутины. Go не отдаёт его напрямую,
// поэтому он берётся из заголовка стека: "goroutine 42 [running]:".
// Нужен только для поиска циклов, вызывается лишь на пути через фабрику.
// Переменная, чтобы в тестах можно было проверить работу без id.
var goroutineID = func() int64 {
	var buf [64]byte

	return parseGoroutineID(buf[:runtime.Stack(buf[:], false)])
}

// parseGoroutineID - разбирает заголовок стека. Если формат не распознан
// (например, изменился в новой версии Go), возвращает unknownGoroutine, а не паникует.
func parseGoroutineID(stack []byte) int64 {
	header, found := bytes.CutPrefix(stack, []byte("goroutine "))
	if !found {
		return unknownGoroutine
	}

	end := bytes.IndexByte(header, ' ')
	if end < 0 {
		return unknownGoroutine
	}

	id, err := strconv.ParseInt(string(header[:end]), 10, 64)
	if err != nil || id <= 0 {
		return unknownGoroutine
	}

	return id
}

// formatPath - "*A -> *B -> *A"
func formatPath(path []key) string {
	names := make([]string, len(path))
	for i, k := range path {
		names[i] = k.String()
	}

	return strings.Join(names, " -> ")
}

// cast - приведение без паники: nil-интерфейс превращается в zero value T
func cast[T any](v interface{}) T {
	t, _ := v.(T)

	return t
}
