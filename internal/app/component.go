package app

import (
	"context"
	"errors"
	"fmt"
)

// Component defines a unit that can be started and stopped by the application
// container.
type Component interface {
	Name() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// Container keeps the collection of components and drives their lifecycle.
type Container struct {
	comps []Component
}

// NewContainer constructs the container with no registered components.
func NewContainer() *Container { return &Container{} }

// Add appends one or multiple components into the container.
func (c *Container) Add(comp ...Component) { c.comps = append(c.comps, comp...) }

// StartAll walks over every registered component and starts it, collecting the
// errors if any component fails.
func (c *Container) StartAll(ctx context.Context) error {
	var errs []error
	for _, comp := range c.comps {
		if err := comp.Start(ctx); err != nil {
			errs = append(errs, fmt.Errorf("%s start failed: %w", comp.Name(), err))
		}
	}
	return errors.Join(errs...)
}

// StopAll stops registered components in the reverse order to guarantee
// dependencies are shut down gracefully.
func (c *Container) StopAll(ctx context.Context) error {
	var errs []error
	for i := len(c.comps) - 1; i >= 0; i-- {
		if err := c.comps[i].Stop(ctx); err != nil {
			errs = append(errs, fmt.Errorf("%s stop failed: %w", c.comps[i].Name(), err))
		}
	}
	return errors.Join(errs...)
}
