// Package domain porte le vocabulaire du stockage d'objets, sans dépendance.
package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
)

// Key est l'emplacement d'un objet dans le magasin.
//
// Distincte du nom demandé par l'appelant : le nom vient de l'extérieur, la clé
// est dérivée et sûre. Les confondre est la faille de traversée de répertoire.
type Key string

// String rend la clé sous forme brute.
func (k Key) String() string { return string(k) }

// Object est un objet à stocker.
type Object struct {
	// Name est le nom demandé par l'appelant. Il vient d'un formulaire, donc d'un
	// inconnu : il n'est JAMAIS utilisé tel quel comme chemin.
	Name string
	// ContentType est déclaré par l'appelant, donc jamais digne de confiance pour
	// une décision de sécurité. Un pilote qui le renvoie en en-tête doit le
	// restreindre à une liste blanche.
	ContentType string
	Content     io.Reader
}

// Located est le résultat d'un stockage réussi.
type Located struct {
	Key Key
	// URL est l'adresse de lecture. Sa forme dépend du pilote — chemin servi par
	// l'application pour `disk`, URL du fournisseur pour un magasin d'objets.
	URL string
}

// ErrUnsafeName refuse un nom qui tenterait de sortir du magasin.
var ErrUnsafeName = errors.New("nom d'objet refusé")

// ErrEmptyContent refuse un objet sans contenu.
var ErrEmptyContent = errors.New("objet sans contenu")

// IsValid indique si l'objet porte le minimum exploitable.
func (o Object) IsValid() bool { return o.Name != "" && o.Content != nil }

// SafeKey dérive une clé de stockage sûre du nom demandé.
//
// # C'est la fonction de sécurité du module
//
// Elle est PURE et vit dans le domaine précisément pour ça : elle se teste
// exhaustivement, sans disque, sans réseau, et tous les pilotes l'appellent. Un
// pilote qui construirait sa clé lui-même rouvrirait la faille pour lui seul.
//
// Trois propriétés, chacune pour une raison :
//
//  1. **Le nom est haché.** La clé ne contient donc aucun fragment de chemin
//     fourni par l'appelant : `../../etc/passwd` ne peut pas survivre au hachage.
//  2. **La clé est répartie sur deux niveaux.** Un répertoire plat à cent mille
//     entrées devient impraticable sur la plupart des systèmes de fichiers.
//  3. **Le nom de base est conservé en suffixe**, nettoyé. Un objet reste
//     identifiable par un humain qui inspecte le magasin — sans quoi un incident
//     se diagnostique à l'aveugle.
func SafeKey(name string) (Key, error) {
	base := path.Base(path.Clean("/" + strings.ReplaceAll(name, "\\", "/")))
	if base == "." || base == ".." || base == "/" || base == "" {
		return "", fmt.Errorf("%w: %q ne désigne aucun fichier", ErrUnsafeName, name)
	}

	sum := sha256.Sum256([]byte(name))
	digest := hex.EncodeToString(sum[:])
	return Key(fmt.Sprintf("%s/%s/%s-%s", digest[0:2], digest[2:4], digest[4:16], base)), nil
}

// IsWithin indique si une clé reste dans les limites du magasin.
//
// Appelée à la LECTURE, où la clé vient de l'extérieur — d'une URL, donc d'un
// inconnu — et n'a pas été dérivée par SafeKey. Sans cette vérification, un `..`
// dans une clé permettrait de lire n'importe quel fichier de l'hôte.
func IsWithin(key Key) bool {
	raw := strings.ReplaceAll(key.String(), "\\", "/")
	if raw == "" || strings.HasPrefix(raw, "/") {
		return false
	}
	cleaned := path.Clean(raw)
	return cleaned == raw && !strings.HasPrefix(cleaned, "../") && cleaned != ".."
}
