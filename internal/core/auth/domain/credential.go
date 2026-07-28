package domain

import "fmt"

// Credential porte une identité ET le condensé de son secret.
//
// # Pourquoi ce type existe plutôt que deux valeurs de retour
//
// La règle d'architecture borne les fonctions du noyau à DEUX valeurs de retour,
// et un port qui rendrait `(Identity, string, error)` serait refusé. Elle a
// raison ici pour une raison propre au sujet : deux `string` de suite —
// « le sujet » et « le condensé » — s'inversent silencieusement, et l'inversion
// produirait une comparaison qui réussit toujours.
//
// # Ce type ne se journalise jamais
//
// Il implémente `Stringer` et `GoStringer` pour MASQUER le condensé. Un condensé
// Argon2id n'est pas un mot de passe, mais il se casse hors ligne : le publier
// dans un journal transforme une fuite de logs en fuite de comptes.
//
// Les deux implémentations sont nécessaires, pas redondantes : `%v` passe par
// `String()`, `%#v` par `GoString()`. Il a fallu un test pour découvrir que
// couvrir l'un laissait l'autre fuiter.
type Credential struct {
	identity   Identity
	secretHash string
}

// NewCredential assemble une identité et le condensé de son secret.
func NewCredential(identity Identity, secretHash string) (Credential, error) {
	if identity.ID == "" {
		return Credential{}, fmt.Errorf("%w: l'identité est obligatoire", ErrIncomplete)
	}
	if secretHash == "" {
		return Credential{}, fmt.Errorf("%w: le condensé du secret est obligatoire", ErrIncomplete)
	}
	return Credential{identity: identity, secretHash: secretHash}, nil
}

// Identity rend l'identité portée.
func (c Credential) Identity() Identity { return c.identity }

// SecretHash rend le condensé, POUR COMPARAISON uniquement.
//
// Nommée explicitement plutôt qu'exposée en champ : un accès au condensé se voit
// alors en relecture, et se cherche en une commande.
func (c Credential) SecretHash() string { return c.secretHash }

// String masque le condensé. Voir la documentation du type.
func (c Credential) String() string {
	return "Credential{identity: " + string(c.identity.ID) + ", secretHash: ***}"
}

// GoString masque le condensé sous `%#v` aussi.
func (c Credential) GoString() string { return c.String() }
