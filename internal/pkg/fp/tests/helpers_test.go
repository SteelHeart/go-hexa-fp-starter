// Package tests contient les tests en BOÎTE NOIRE des primitives fonctionnelles :
// ils n'utilisent que l'API publique, exactement comme un appelant.
//
// Convention du dépôt (rules/tests.md) : `{paquet}/tests/` pour la boîte noire,
// `{paquet}/internal_test.go` pour les identifiants non exportés. Un fichier par
// test — le nom du fichier dit ce qui est vérifié, sans avoir à l'ouvrir.
package tests

import "strconv"

// Trois fonctions pures, pour composer.
func double(n int) int       { return n * 2 }
func incremente(n int) int   { return n + 1 }
func versTexte(n int) string { return strconv.Itoa(n) }

// pair est le prédicat de référence des tests sur les tranches.
func pair(n int) bool { return n%2 == 0 }
