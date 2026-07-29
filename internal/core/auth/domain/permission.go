package domain

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// permissionShape constrains the shape of a permission.
//
// `domain.resource.action`, in lowercase. Exactly three segments — neither two,
// nor four.
//
// The constraint is not cosmetic: a permission is a string, so nothing prevents
// writing `admin` on one side and `Admin` on the other. Both would coexist, one
// would grant, the other would not, and the defect would show up as a "they
// have the right but it refuses" — impossible to diagnose without comparing two
// strings by eye.
var permissionShape = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*){2}$`)

// Permission names a precise action, never a role.
//
// This is decision 4 of ADR 017, carried by the TYPE: the authorisation port
// only accepts a Permission, so the compiler refuses `authorize(id, role)`.
// With roles tested in the code, adding a role is a deployment; with
// permissions, it is a piece of data.
type Permission struct{ value string }

// NewPermission validates and normalises a permission.
//
// This is the ONLY construction path: the field is private, so an invalid
// Permission cannot exist outside this package.
func NewPermission(raw string) (Permission, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if !permissionShape.MatchString(normalized) {
		return Permission{}, fmt.Errorf(
			"%w: permission %q — expected `domain.resource.action` in lowercase", ErrIncomplete, raw)
	}
	return Permission{value: normalized}, nil
}

// String returns the normalised permission.
func (p Permission) String() string { return p.value }

// IsZero reports a permission that was never built.
func (p Permission) IsZero() bool { return p.value == "" }

// Role is a name that CARRIES permissions.
//
// The role exists in order to be administered — you give someone "accountant",
// not seventeen permissions one by one. It never appears in an authorisation
// decision.
type Role struct {
	Name        string
	Permissions []Permission
}

// NewRole validates a role and sorts its permissions.
//
// Sorting makes the equality of two roles observable and the messages
// comparable from one run to the next — a map's order changes on every walk.
//
// Duplicates are ABSORBED rather than refused: the same permission twice
// expresses the same intent, and refusing would fail a data import over a
// harmless redundancy.
func NewRole(name string, permissions []Permission) (Role, error) {
	trimmed := strings.ToLower(strings.TrimSpace(name))
	if trimmed == "" {
		return Role{}, fmt.Errorf("%w: the role name is mandatory", ErrIncomplete)
	}

	unique := make(map[string]Permission, len(permissions))
	for _, permission := range permissions {
		if permission.IsZero() {
			return Role{}, fmt.Errorf("%w: role %q carries a permission that was never built", ErrIncomplete, trimmed)
		}
		unique[permission.String()] = permission
	}

	kept := make([]Permission, 0, len(unique))
	for _, permission := range unique {
		kept = append(kept, permission)
	}
	sort.Slice(kept, func(i, j int) bool { return kept[i].String() < kept[j].String() })

	return Role{Name: trimmed, Permissions: kept}, nil
}

// Grants reports whether the role grants a permission.
//
// Exact comparison, no wildcard. A `billing.*` would save a few lines of
// configuration and make it impossible to answer "who can cancel an invoice?"
// otherwise than by running the code.
func (r Role) Grants(permission Permission) bool {
	for _, granted := range r.Permissions {
		if granted == permission {
			return true
		}
	}
	return false
}
