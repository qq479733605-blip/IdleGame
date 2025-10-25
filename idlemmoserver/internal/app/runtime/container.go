package runtime

import (
	"fmt"
	"reflect"
	"sync"
)

// ServiceKey uniquely identifies a dependency registered inside the runtime container.
type ServiceKey string

// Container is a lightweight dependency registry that keeps modules decoupled via
// explicit service keys. Modules can publish capabilities and resolve the
// dependencies they need at runtime.
type Container struct {
	mu       sync.RWMutex
	services map[ServiceKey]interface{}
}

// NewContainer creates an empty runtime container instance.
func NewContainer() *Container {
	return &Container{services: make(map[ServiceKey]interface{})}
}

// Provide registers a service instance under the given key. It returns an error if a
// service has already been registered for that key.
func (c *Container) Provide(key ServiceKey, svc interface{}) error {
	if key == "" {
		return fmt.Errorf("container: empty service key")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.services[key]; exists {
		return fmt.Errorf("container: service %q already registered", key)
	}

	c.services[key] = svc
	return nil
}

// MustProvide registers the service and panics if the key is already present. This is
// helpful during application bootstrap where failures should abort immediately.
func (c *Container) MustProvide(key ServiceKey, svc interface{}) {
	if err := c.Provide(key, svc); err != nil {
		panic(err)
	}
}

func (c *Container) resolve(key ServiceKey) (interface{}, error) {
	c.mu.RLock()
	svc, ok := c.services[key]
	c.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("container: service %q not found", key)
	}
	return svc, nil
}

// Resolve returns the registered service cast to the requested type. If the service is
// missing or has an unexpected type an error is returned.
func Resolve[T any](c *Container, key ServiceKey) (T, error) {
	var zero T

	svc, err := c.resolve(key)
	if err != nil {
		return zero, err
	}

	typed, ok := svc.(T)
	if !ok {
		expected := reflect.TypeOf((*T)(nil)).Elem()
		return zero, fmt.Errorf("container: service %q has type %T (expected %s)", key, svc, expected)
	}

	return typed, nil
}

// MustResolve returns the registered service or panics if it cannot be resolved.
func MustResolve[T any](c *Container, key ServiceKey) T {
	svc, err := Resolve[T](c, key)
	if err != nil {
		panic(err)
	}
	return svc
}
