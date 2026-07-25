// Package domain porte le vocabulaire de l'idempotence, sans dépendance.
//
// # Pourquoi ce module existe
//
// Dès qu'un frontend mobile est servi, le réseau est instable : le client renvoie
// sa requête sans savoir si la première a abouti. Sans clé d'idempotence, un POST
// rejoué crée deux ressources — et le doublon est découvert par l'utilisateur,
// pas par le service.
package domain

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
)

// Key est la clé fournie par l'appelant. Deux requêtes portant la même clé
// désignent la MÊME intention.
type Key string

// String rend la clé sous forme brute, pour les pilotes qui l'écrivent.
func (k Key) String() string { return string(k) }

// Status est l'état d'une réservation. Déclaré ici et non dans un pilote : tous
// les pilotes doivent écrire le même vocabulaire, sinon un changement de pilote
// rendrait les mémorisations existantes illisibles.
type Status string

const (
	// StatusInFlight marque une opération commencée et non résolue.
	StatusInFlight Status = "in_flight"
	// StatusDone marque une opération achevée dont la réponse est mémorisée.
	StatusDone Status = "done"
)

// ErrConflict signale une clé réutilisée avec une empreinte différente.
//
// C'est un défaut du client : la même clé doit désigner la même requête, sinon
// la garantie ne veut plus rien dire. On refuse plutôt que de deviner laquelle
// des deux requêtes est la bonne.
var ErrConflict = errors.New("clé d'idempotence réutilisée avec une requête différente")

// ErrInFlight signale qu'une requête identique est déjà en cours.
//
// L'appelant doit réessayer plus tard, jamais exécuter. C'est le refus qui rend
// la garantie vraie en concurrence.
var ErrInFlight = errors.New("requête identique déjà en cours")

// ErrIncomplete refuse une requête sans clé ou sans empreinte.
//
// Une clé vide serait partagée par TOUS les appelants : le premier rejeu de
// n'importe qui masquerait l'opération de n'importe quel autre. Un pilote qui
// accepterait la clé vide transformerait une protection en fuite de données.
var ErrIncomplete = errors.New("requête d'idempotence incomplète")

// ErrNotReserved signale une mémorisation sans réservation vivante.
//
// Arrive quand la réservation a expiré pendant l'opération. L'appelant a déjà
// exécuté : il doit le journaliser, jamais annuler. Cette erreur dit « la
// mémorisation a échoué », pas « l'opération a échoué ».
var ErrNotReserved = errors.New("aucune réservation vivante pour cette clé")

// Request est ce qu'une clé engage.
type Request struct {
	Key Key
	// Fingerprint est l'empreinte de la charge utile, calculée par Fingerprint.
	Fingerprint string
}

// IsComplete indique si la requête porte le minimum exploitable.
func (r Request) IsComplete() bool {
	return r.Key != "" && r.Fingerprint != ""
}

// Reservation est l'issue d'une réservation obtenue.
//
// # Pourquoi une struct à deux champs et non fp.Option
//
// Les ports d'un module noyau ne dépendent que de leur domaine
// (.arch-go.yml §4e) : ils ne peuvent pas nommer `fp.Option`. Le gain est réel —
// `Replayed` se lit à l'appel, là où un `Option` vide se confond avec « réponse
// mémorisée mais vide ».
type Reservation struct {
	// Replayed à true signifie : NE PAS exécuter, retourner Response telle quelle.
	Replayed bool
	// Response est la réponse mémorisée du premier appel. Nil si Replayed est faux.
	Response []byte
}

// Fingerprint calcule l'empreinte d'une charge utile.
//
// Déterministe : encoding/json ordonne les clés d'une map, donc deux appels sur
// la même valeur rendent la même empreinte. Un changement de forme de la charge
// change l'empreinte, ce qui est exactement le but — la clé ne doit pas couvrir
// deux requêtes différentes.
func Fingerprint(payload any) string {
	raw, err := json.Marshal(payload)
	if err != nil {
		// Une charge non sérialisable est un défaut de programmation, pas une
		// erreur d'exécution. On rend une empreinte qui ne collisionne pas plutôt
		// que de propager une erreur sur un chemin purement défensif.
		raw = []byte(fmt.Sprintf("%#v", payload))
	}
	sum := sha256.Sum256(raw)
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
