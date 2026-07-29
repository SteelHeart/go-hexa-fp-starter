package config

import (
	"errors"
	"fmt"
	"slices"
)

// errorf is a local shorthand: validation messages are complete sentences,
// without any wrapping.
func errorf(format string, args ...any) error {
	return fmt.Errorf(format, args...) //nolint:err113 // validation message, not an error to compare
}

// appendUnlessOneOf adds a problem if the value is not in the list.
//
// Deny by default, all the way into validation: an unforeseen value is refused,
// never interpreted as "the closest one".
func appendUnlessOneOf(problems []error, field, value string, allowed ...string) []error {
	if slices.Contains(allowed, value) {
		return problems
	}
	return append(problems, errors.New(
		field+"="+quote(value)+" unknown (expected: "+join(allowed)+")"))
}

func quote(value string) string { return `"` + value + `"` }

func join(items []string) string {
	out := ""
	for i, item := range items {
		if i > 0 {
			out += ", "
		}
		out += item
	}
	return out
}
