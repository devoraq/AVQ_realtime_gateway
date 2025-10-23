package app

import (
	"context"
	"errors"
	"fmt"
)

type Component interface {
	Name() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type Container struct {
	comps []Component
}

func NewContainer() *Container { return &Container{} }

func (c *Container) Add(comp ...Component) { c.comps = append(c.comps, comp...) }

func (c *Container) StartAll(ctx context.Context) error {
	var errs []error
	for _, comp := range c.comps {
		if err := comp.Start(ctx); err != nil {
			errs = append(errs, fmt.Errorf("%s start failed: %w", comp.Name(), err))
		}
	}
	return errors.Join(errs...)
}

func (c *Container) StopAll(ctx context.Context) error {
	var first error
	for i := len(c.comps) - 1; i >= 0; i-- {
		if err := c.comps[i].Stop(ctx); err != nil {
			first = fmt.Errorf("%s stop failed: %w", c.comps[i].Name(), err)
		}
	}
	return first
}
