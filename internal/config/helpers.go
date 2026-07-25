package config

import (
	"errors"
	"fmt"
	"slices"
)

// errorf est un raccourci local : les messages de validation sont des phrases
// complètes, sans enveloppe.
func errorf(format string, args ...any) error {
	return fmt.Errorf(format, args...) //nolint:err113 // message de validation, pas une erreur à comparer
}

// appendUnlessOneOf ajoute un problème si la valeur n'est pas dans la liste.
//
// Deny par défaut jusque dans la validation : une valeur non prévue est refusée,
// jamais interprétée comme « la plus proche ».
func appendUnlessOneOf(problems []error, field, value string, allowed ...string) []error {
	if slices.Contains(allowed, value) {
		return problems
	}
	return append(problems, errors.New(
		field+"="+quote(value)+" inconnu (attendu: "+join(allowed)+")"))
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
