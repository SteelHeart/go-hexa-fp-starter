package domain

import (
	"crypto/subtle"
	"fmt"
	"time"
)

// tokenMinLen borne la longueur d'un jeton opaque.
//
// 43 caractères, soit 32 octets en base64 sans remplissage. En dessous, l'entropie
// devient devinable ; la borne est ici pour que le domaine REFUSE un jeton
// fabriqué court, quel que soit le port qui l'a produit.
const tokenMinLen = 43

// Token est un jeton OPAQUE — une chaîne aléatoire, pas un JWT.
//
// # Pourquoi opaque, et pas signé (ADR 017 § 2 bis)
//
// Le seul avantage réel d'un jeton signé est de valider SANS toucher au magasin.
// Or `Authorize` y va de toute façon, à chaque appel : c'est la décision 1, et
// elle est ce qui rend la révocation immédiate. On paierait donc la gestion de
// clés, leur rotation, et toute la famille de failles propre aux jetons signés —
// algorithme `none` accepté, confusion HMAC/RSA, expiration non vérifiée — pour
// un gain déjà abandonné.
//
// Une chaîne aléatoire comparée à un enregistrement n'a aucune de ces failles.
type Token struct{ value string }

// NewToken valide un jeton produit par un port d'aléa.
func NewToken(raw string) (Token, error) {
	if len(raw) < tokenMinLen {
		return Token{}, fmt.Errorf(
			"%w: un jeton doit faire au moins %d caractères", ErrIncomplete, tokenMinLen)
	}
	return Token{value: raw}, nil
}

// String rend le jeton brut. À n'appeler que pour le transmettre au client ou au
// pilote — jamais pour le journaliser.
func (t Token) String() string { return t.value }

// IsZero indique un jeton non construit.
func (t Token) IsZero() bool { return t.value == "" }

// Equals compare deux jetons en TEMPS CONSTANT.
//
// `==` s'arrête au premier octet différent, donc la durée de la comparaison
// révèle combien de caractères de tête sont corrects. C'est l'attaque par mesure
// de temps, et elle est praticable sur un réseau local. `subtle.ConstantTimeCompare`
// coûte la même chose et supprime le canal.
func (t Token) Equals(other Token) bool {
	return subtle.ConstantTimeCompare([]byte(t.value), []byte(other.value)) == 1
}

// Session est un jeton émis, rattaché à une identité et daté.
//
// Les permissions n'y figurent PAS, et c'est tout le sujet de l'ADR 017 : le
// jeton authentifie, il n'autorise pas. Les y mettre créerait une fenêtre pendant
// laquelle un accès révoqué fonctionne encore — et le jour où l'on révoque, c'est
// qu'on est pressé.
type Session struct {
	Token     Token
	Identity  IdentityID
	IssuedAt  time.Time
	ExpiresAt time.Time
}

// NewSession construit une session bornée dans le temps.
//
// Une durée nulle ou négative est REFUSÉE : une session sans expiration est une
// session éternelle, et personne ne décide cela par erreur.
func NewSession(token Token, identity IdentityID, now time.Time, ttl time.Duration) (Session, error) {
	switch {
	case token.IsZero():
		return Session{}, fmt.Errorf("%w: le jeton est obligatoire", ErrIncomplete)
	case identity == "":
		return Session{}, fmt.Errorf("%w: l'identité est obligatoire", ErrIncomplete)
	case ttl <= 0:
		return Session{}, fmt.Errorf("%w: la durée de session doit être strictement positive", ErrIncomplete)
	}

	issued := now.UTC()
	return Session{
		Token:     token,
		Identity:  identity,
		IssuedAt:  issued,
		ExpiresAt: issued.Add(ttl),
	}, nil
}

// Expired indique si la session a expiré à l'instant donné.
//
// L'instant est un PARAMÈTRE : le domaine ne lit pas l'horloge. C'est ce qui
// permet de tester l'expiration sans attendre.
//
// La borne est stricte — une session expire À sa date, pas après. Un `>` laisserait
// passer la dernière milliseconde, ce qui n'a aucune importance pratique et rend
// le test dépendant d'une comparaison qu'on relit deux fois.
func (s Session) Expired(now time.Time) bool {
	return !now.UTC().Before(s.ExpiresAt)
}
