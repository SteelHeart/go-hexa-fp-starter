// Package file implements dynamic configuration from the versioned values of
// the repository.
//
// The values come from the `options` of the module in `config/modules.yaml`,
// and not from a separate file: the loader refuses any unknown key in
// `config/*.yaml` (`KnownFields(true)`), so a new file would break startup
// without modifying the `Config` type. The options, for their part, are
// deliberately untyped.
//
// # NON-GUARANTEES — to be read before using it
//
//   - **NOT changeable at run time.** This is THE non-guarantee of the driver,
//     and it contradicts the reason the module exists. Changing a flag requires
//     a redeployment. In development, `config/local.yaml` — not versioned —
//     overrides any value by merging, without touching the repository.
//   - **`Set` always refuses** (`domain.ErrReadOnly`). Writing into a versioned
//     file at run time would create a divergence between the repository and
//     what is running, which the next deployment would overwrite without
//     warning.
//
// Suitable in development, in test, and in production as long as flags change
// at the rhythm of deployments. As soon as a feature must be switched off
// without redeploying, move to the `postgres` driver.
package file

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/dynconf/domain"
)

// Store is the read-only store.
//
// No lock: the map is built once then never modified. It is immutability that
// makes the store usable from several goroutines, not a mutex.
type Store struct {
	values map[string]string
}

// New builds the store from the options of the driver.
//
// A non-scalar value refuses to start: `flags: {a: {b: 1}}` is a typo, not an
// intention, and letting it through would give a silently inactive flag.
func New(flags, settings map[string]any) (*Store, error) {
	values := make(map[string]string, len(flags)+len(settings))
	if err := fill(values, domain.KindFlag, "flags", flags); err != nil {
		return nil, err
	}
	if err := fill(values, domain.KindSetting, "settings", settings); err != nil {
		return nil, err
	}
	return &Store{values: values}, nil
}

// fill converts a group of options into qualified textual values.
func fill(dst map[string]string, kind domain.Kind, label string, src map[string]any) error {
	for key, raw := range src {
		text, err := scalar(raw)
		if err != nil {
			return fmt.Errorf("options.%s.%s: %w", label, key, err)
		}
		dst[domain.Qualify(kind, key)] = text
	}
	return nil
}

// errNotScalar refuses a value that is not a value.
var errNotScalar = errors.New("non-scalar value")

// scalar returns the textual form of an option.
//
// Everything goes through text: that is what allows the `postgres` driver,
// which reads a `text` column, to be substitutable for this one without the
// caller knowing.
func scalar(raw any) (string, error) {
	switch value := raw.(type) {
	case string:
		return value, nil
	case bool:
		return strconv.FormatBool(value), nil
	case int:
		return strconv.Itoa(value), nil
	case int64:
		return strconv.FormatInt(value, 10), nil
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64), nil
	default:
		return "", fmt.Errorf("%w (%T)", errNotScalar, raw)
	}
}

// Flag implements ports.IsEnabled.
func (s *Store) Flag(_ context.Context, key domain.FlagKey) bool {
	raw, found := s.values[domain.Qualify(domain.KindFlag, string(key))]
	if !found {
		return false
	}
	return domain.ParseFlag(raw)
}

// Setting implements ports.GetSetting.
func (s *Store) Setting(_ context.Context, key domain.SettingKey) domain.Setting {
	raw, found := s.values[domain.Qualify(domain.KindSetting, string(key))]
	return domain.Setting{Value: raw, Found: found}
}

// Set implements ports.Set by refusing.
//
// It still checks the validity of the change: a caller who is going to fix
// their driver must first learn that their change was malformed.
func (s *Store) Set(_ context.Context, change domain.Change) error {
	if !change.IsValid() {
		return fmt.Errorf("%w: %s", domain.ErrInvalidChange, change.Describe())
	}
	return fmt.Errorf("%w: %s", domain.ErrReadOnly, change.Describe())
}

// Invalidate implements ports.Invalidate. Without effect: nothing is cached,
// since everything is already in memory and immutable.
func (s *Store) Invalidate() {}
