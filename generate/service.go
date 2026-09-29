package generate

import (
	"errors"
	"fmt"
)

//go:generate mockgen -source=service.go -destination=mocks/service_mock.go -package=mocks

// Service is the kind of interface that is worth mocking: it crosses a
// package boundary in production code, so tests can replace it with the
// generated double.
type Service interface {
	Fetch(id string) (string, error)
}

// ErrNotFound is returned by Cache when a lookup misses.
var ErrNotFound = errors.New("generate: not found")

// Cache is ordinary code that depends on Service. It is the production side
// of the example; the test drives it with the generated mock.
type Cache struct {
	svc Service
}

// NewCache wires a Cache to a Service.
func NewCache(svc Service) *Cache {
	return &Cache{svc: svc}
}

// Label returns the label for id, or ErrNotFound when the service does not
// know it.
func (c *Cache) Label(id string) (string, error) {
	value, err := c.svc.Fetch(id)
	if err != nil {
		return "", fmt.Errorf("fetch %q: %w", id, err)
	}
	if value == "" {
		return "", ErrNotFound
	}
	return value, nil
}
