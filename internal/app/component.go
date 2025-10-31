package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/DENFNC/devPractice/internal/adapters/outbound/config"
	"github.com/DENFNC/devPractice/pkg/retry"
)

// Component описывает компонент, жизненным циклом которого управляет контейнер.
type Component interface {
	Name() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// Container хранит набор компонентов и управляет их жизненным циклом.
type Container struct {
	comps map[string]Component
	log   *slog.Logger
	cfg   *config.Config
}

// NewContainer создаёт пустой контейнер без зарегистрированных компонентов.
func NewContainer(log *slog.Logger, cfg *config.Config) *Container {
	return &Container{
		comps: make(map[string]Component),
		log:   log,
		cfg:   cfg,
	}
}

// Add добавляет один или несколько компонентов в контейнер.
func (c *Container) Add(comps ...Component) {
	for _, comp := range comps {
		c.comps[comp.Name()] = comp
	}
}

// StartAll последовательно запускает компоненты, накапливая ошибки запуска.
func (c *Container) StartAll(ctx context.Context) error {
	var errs []error
	for _, comp := range c.comps {
		component := c.comps[comp.Name()]

		err := retry.Do(ctx, c.cfg, func(ctx context.Context) error {
			return component.Start(ctx)
		})

		if err != nil {
			errs = append(errs, fmt.Errorf("%s start failed: %w", component.Name(), err))
		}
	}
	return errors.Join(errs...)
}

// StopAll останавливает компоненты в обратном порядке, обеспечивая корректное завершение зависимостей.
func (c *Container) StopAll(ctx context.Context) error {
	var errs []error
	for i, comp := range c.comps {
		if err := c.comps[comp.Name()].Stop(ctx); err != nil {
			errs = append(errs, fmt.Errorf("%s stop failed: %w", c.comps[i].Name(), err))
		}
	}
	return errors.Join(errs...)
}

func (c *Container) Get(name string) (any, error) {
	component, ok := c.comps[name]
	if !ok {
		return nil, fmt.Errorf("component don`t exist")
	}
	return component, nil
}
