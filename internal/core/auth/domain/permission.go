package domain

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// permissionShape contraint la forme d'une permission.
//
// `domaine.ressource.action`, en minuscules. Trois segments exactement — ni deux,
// ni quatre.
//
// La contrainte n'est pas cosmétique : une permission est une chaîne, donc rien
// n'empêche d'écrire `admin` d'un côté et `Admin` de l'autre. Les deux
// coexisteraient, l'une accorderait, l'autre pas, et le défaut se manifesterait
// comme un « il a le droit mais ça refuse » — impossible à diagnostiquer sans
// comparer deux chaînes à l'œil.
var permissionShape = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*){2}$`)

// Permission nomme une action précise, jamais un rôle.
//
// C'est la décision 4 de l'ADR 017, portée par le TYPE : le port d'autorisation
// n'accepte qu'une Permission, donc le compilateur refuse `authorize(id, role)`.
// Avec des rôles testés dans le code, ajouter un rôle est un déploiement ; avec
// des permissions, c'est une donnée.
type Permission struct{ value string }

// NewPermission valide et normalise une permission.
//
// C'est le SEUL chemin de construction : le champ est privé, donc une Permission
// invalide ne peut pas exister hors de ce paquet.
func NewPermission(raw string) (Permission, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if !permissionShape.MatchString(normalized) {
		return Permission{}, fmt.Errorf(
			"%w: permission %q — attendu `domaine.ressource.action` en minuscules", ErrIncomplete, raw)
	}
	return Permission{value: normalized}, nil
}

// String rend la permission normalisée.
func (p Permission) String() string { return p.value }

// IsZero indique une permission non construite.
func (p Permission) IsZero() bool { return p.value == "" }

// Role est un nom qui PORTE des permissions.
//
// Le rôle existe pour être administré — on donne « comptable » à quelqu'un, pas
// dix-sept permissions une par une. Il n'apparaît jamais dans une décision
// d'autorisation.
type Role struct {
	Name        string
	Permissions []Permission
}

// NewRole valide un rôle et trie ses permissions.
//
// Le tri rend l'égalité de deux rôles observable et les messages comparables
// d'une exécution à l'autre — un ordre de map change à chaque parcours.
//
// Les doublons sont ABSORBÉS plutôt que refusés : deux fois la même permission
// exprime la même intention, et refuser ferait échouer un import de données pour
// une redondance sans conséquence.
func NewRole(name string, permissions []Permission) (Role, error) {
	trimmed := strings.ToLower(strings.TrimSpace(name))
	if trimmed == "" {
		return Role{}, fmt.Errorf("%w: le nom du rôle est obligatoire", ErrIncomplete)
	}

	unique := make(map[string]Permission, len(permissions))
	for _, permission := range permissions {
		if permission.IsZero() {
			return Role{}, fmt.Errorf("%w: le rôle %q porte une permission non construite", ErrIncomplete, trimmed)
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

// Grants indique si le rôle accorde une permission.
//
// Comparaison exacte, sans joker. Un `billing.*` ferait gagner quelques lignes de
// configuration et rendrait impossible de répondre à « qui peut annuler une
// facture ? » autrement qu'en exécutant le code.
func (r Role) Grants(permission Permission) bool {
	for _, granted := range r.Permissions {
		if granted == permission {
			return true
		}
	}
	return false
}
