// Package storage is the core module of object storage.
//
// Composition root of the module: the only place that knows the drivers.
//
// # Why only one driver is shipped
//
// An S3, GCS or Azure Blob driver pulls an SDK of several tens of megabytes
// into the dependency graph of EVERY generated project, including those that
// will never go near an object store. The driver rules settle it: heavy drivers
// are separate Go modules (issue #22). They are therefore not "forgotten", they
// are elsewhere — and `knownDrivers` only lists what exists.
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/storage/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/storage/drivers/disk"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/storage/ports"
)

// Name is the name of the module in config/modules.yaml.
const Name = "storage"

// Names of the drivers of this module.
//
// They exist so that `Catalog` and the `switch` of `New` share the SAME
// identifier. This is what makes divergence between the two IMPOSSIBLE, where
// ADR 014 only promised to make it improbable — the compiler refuses a
// constant that does not exist, a misspelt literal goes through.
//
// The `goconst` linter reported the repetition as soon as the catalogue
// arrived. It was right, and for a stronger reason than its own.
const (
	driverDisk = "disk"
)

// Option keys read by the disk driver.
//
// Declared here, REFERENCED by the catalogue: two separate lists would end up
// diverging, and an admitted option that nobody reads is never noticed.
const (
	OptionBaseDir = "base_dir"
	OptionBaseURL = "base_url"
)

// Default values of the disk driver.
const (
	defaultBaseDir = "var/storage"
	defaultBaseURL = "/files"
)

// Module exposes the ports of storage.
type Module struct {
	Put    ports.Put
	Get    ports.Get
	Delete ports.Delete
}

// Deps carries the dependencies of the drivers.
//
// Empty today: the `disk` driver claims nothing beyond its configuration. The
// type exists so that adding a driver that claims a client does not change the
// signature of New — and therefore no caller.
type Deps struct{}

// ErrDisabled signals a call to a disabled module.
var ErrDisabled = errors.New("storage module disabled in config/modules.yaml")

var errUnknownDriver = errors.New("unknown storage driver")

// New builds the module according to the configuration.
func New(cfg config.Module, _ Deps) (Module, error) {
	if !cfg.Enabled {
		return disabled(), nil
	}

	switch cfg.Driver {
	case driverDisk:
		return fromDisk(cfg)
	default:
		return Module{}, fmt.Errorf("%w: %q", errUnknownDriver, cfg.Driver)
	}
}

// fromDisk builds the local driver.
func fromDisk(cfg config.Module) (Module, error) {
	baseDir, err := cfg.StringOption(OptionBaseDir, defaultBaseDir)
	if err != nil {
		return Module{}, fmt.Errorf("modules.%s.%w", Name, err)
	}
	baseURL, err := cfg.StringOption(OptionBaseURL, defaultBaseURL)
	if err != nil {
		return Module{}, fmt.Errorf("modules.%s.%w", Name, err)
	}
	store, err := disk.New(baseDir, baseURL)
	if err != nil {
		return Module{}, fmt.Errorf("modules.%s: %w", Name, err)
	}
	return Module{Put: store.Put, Get: store.Get, Delete: store.Delete}, nil
}

// disabled returns ports that refuse explicitly.
func disabled() Module {
	return Module{
		Put: func(context.Context, domain.Object) (domain.Located, error) {
			return domain.Located{}, ErrDisabled
		},
		Get:    func(context.Context, domain.Key) (io.ReadCloser, error) { return nil, ErrDisabled },
		Delete: func(context.Context, domain.Key) error { return ErrDisabled },
	}
}
