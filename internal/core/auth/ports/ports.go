// Package ports déclare les contrats de l'authentification.
//
// Ce paquet ne contient QUE des déclarations de types : ni struct, ni fonction,
// ni interface. Un port est un type fonction — la plus petite interface possible,
// donc rien à ségréguer (ADR 003).
//
// # Pourquoi `error` et non `Result[T, domain.Error]`
//
// `internal/core/**` utilise `error`, `internal/modules/**` utilise `Result` :
// la frontière est nette et vérifiable. `auth` est le cas limite — il a une
// taxonomie que les surfaces traduisent en 401, 403 et 422 — et elle passe par
// des sentinelles énumérées, reconnaissables par `errors.Is` (ADR 017).
//
// # Le protocole, en trois appels
//
//	token, err := authenticate(ctx, subject, secret)   // 401 si ErrInvalidCredentials
//	identity, err := verify(ctx, token)                // 401 si ErrTokenUnknown
//	err := authorize(ctx, identity, permission)        // 403 si ErrForbidden
//
// `verify` et `authorize` sont DEUX appels et non un seul, délibérément : le
// jeton authentifie, il n'autorise pas. Les fusionner ramènerait les permissions
// dans le jeton par la porte de derrière.
package ports

import (
	"context"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// ─── Ports primaires : ce que le monde extérieur peut demander ───────────────

// Authenticate échange un secret contre une session.
//
// Erreurs : `domain.ErrIncomplete` si la demande est mal formée,
// `domain.ErrInvalidCredentials` sinon — et JAMAIS de distinction entre « sujet
// inconnu » et « secret faux » : la différence dirait à un attaquant quels comptes
// existent.
type Authenticate = func(
	ctx context.Context,
	subject string,
	secret string,
) (domain.Session, error)

// Verify résout un jeton en identité.
//
// Erreurs : `domain.ErrTokenUnknown` pour un jeton inconnu, expiré OU révoqué —
// les trois se confondent pour l'appelant, et c'est voulu.
type Verify = func(ctx context.Context, token domain.Token) (domain.Identity, error)

// Authorize vérifie qu'une identité détient une permission.
//
// # Le contrat qui porte toute l'ADR 017
//
// L'implémentation interroge l'état PERSISTÉ à chaque appel. Elle ne lit pas de
// revendication portée par un jeton, et ne met rien en cache sans que ce cache
// soit un décorateur explicite et borné.
//
// Le paramètre est une `domain.Permission`, jamais un rôle : le compilateur
// interdit donc `authorize(ctx, id, "admin")`.
//
// Erreurs : `domain.ErrForbidden` si la permission manque.
type Authorize = func(
	ctx context.Context,
	identity domain.IdentityID,
	permission domain.Permission,
) error

// Revoke invalide une session immédiatement.
//
// Révoquer un jeton déjà inconnu n'est PAS une erreur : l'opération est
// idempotente, parce qu'un client qui se déconnecte deux fois n'a rien fait de mal.
type Revoke = func(ctx context.Context, token domain.Token) error

// Register crée une identité et son secret.
//
// Erreurs : `domain.ErrSubjectTaken` si le sujet existe déjà.
type Register = func(ctx context.Context, subject, secret string) (domain.Identity, error)

// DefineRole crée ou remplace un rôle et ses permissions.
//
// Sans ce port, aucune permission ne peut jamais être accordée : `Authorize`
// refuserait tout, et le module serait inerte tout en ayant l'air de fonctionner.
//
// REMPLACE plutôt qu'ajoute : retirer une permission d'un rôle doit être aussi
// simple que d'en ajouter une. Une API qui n'offrirait que l'ajout ferait écrire
// le retrait à la main, donc mal.
type DefineRole = func(ctx context.Context, name string, permissions []string) error

// AssignRoles remplace les rôles d'une identité.
//
// Erreurs : `domain.ErrInvalidCredentials` si l'identité est inconnue.
type AssignRoles = func(ctx context.Context, id domain.IdentityID, roles []string) error

// Deactivate ferme un compte, IMMÉDIATEMENT.
//
// C'est le geste qu'on fait quand on découvre qu'un compte est compromis, et
// c'est donc le seul moment où la latence compte vraiment. L'identité cesse de
// s'authentifier, ses jetons déjà émis cessent de valoir, et ses permissions
// cessent d'être accordées — les trois au prochain appel, sans expiration.
//
// Idempotent : désactiver un compte déjà fermé n'est pas une erreur. Deux
// administrateurs qui réagissent au même incident ne doivent pas s'annuler.
//
// Erreurs : `domain.ErrInvalidCredentials` si l'identité est inconnue.
type Deactivate = func(ctx context.Context, id domain.IdentityID) error

// Reactivate rouvre un compte fermé.
//
// Existe parce que la désactivation est parfois une erreur, et qu'un module qui
// ne saurait que fermer ferait réparer cela à la main dans le magasin — donc mal,
// et sans trace.
//
// Erreurs : `domain.ErrInvalidCredentials` si l'identité est inconnue.
type Reactivate = func(ctx context.Context, id domain.IdentityID) error

// ─── Ports secondaires : ce dont le cœur a besoin du monde ───────────────────

// SaveIdentity persiste une identité et le condensé de son secret.
//
// Contrat d'erreur : `domain.ErrSubjectTaken` si le sujet existe déjà. C'est
// l'implémentation qui tranche, pas le cas d'usage : entre une vérification et
// une écriture il existe une fenêtre que deux demandes simultanées franchissent
// toutes les deux.
type SaveIdentity = func(ctx context.Context, credential domain.Credential) error

// FindBySubject retrouve une identité et son condensé.
//
// Contrat d'erreur : `domain.ErrInvalidCredentials` pour un sujet inconnu — et
// non une erreur « introuvable ». Le cas d'usage doit être INCAPABLE de
// distinguer « ce sujet n'existe pas » de « le secret est faux », sinon il
// finirait par le dire au client.
type FindBySubject = func(ctx context.Context, subject domain.Subject) (domain.Credential, error)

// FindIdentity retrouve une identité par son identifiant.
type FindIdentity = func(ctx context.Context, id domain.IdentityID) (domain.Identity, error)

// SaveSession persiste une session émise.
type SaveSession = func(ctx context.Context, session domain.Session) error

// FindSession retrouve une session par son jeton.
//
// Contrat d'erreur : `domain.ErrTokenUnknown`.
type FindSession = func(ctx context.Context, token domain.Token) (domain.Session, error)

// DeleteSession révoque une session. Idempotente.
type DeleteSession = func(ctx context.Context, token domain.Token) error

// SaveRole persiste un rôle et ses permissions.
type SaveRole = func(ctx context.Context, role domain.Role) error

// BindRoles remplace les rôles d'une identité dans le magasin.
type BindRoles = func(ctx context.Context, id domain.IdentityID, roles []string) error

// UpdateIdentity remplace l'identité retenue, sans toucher au condensé du secret.
//
// Le condensé n'est PAS un paramètre, délibérément : une signature qui le
// reprendrait obligerait chaque appelant à le transporter, donc à le lire, donc à
// pouvoir le journaliser. Fermer un compte n'a aucune raison de faire circuler un
// condensé.
//
// Contrat d'erreur : `domain.ErrInvalidCredentials` si l'identité est inconnue.
type UpdateIdentity = func(ctx context.Context, identity domain.Identity) error

// Grants interroge l'état PERSISTÉ des permissions d'une identité.
//
// Rend un booléen et non une erreur : « ne détient pas la permission » n'est pas
// une panne, c'est une réponse. C'est le cas d'usage qui la traduit en
// `domain.ErrForbidden`.
type Grants = func(ctx context.Context, id domain.IdentityID, permission domain.Permission) bool

// HashSecret produit le condensé d'un secret.
//
// C'est un port PRÉCISÉMENT parce que le hachage est un effet coûteux et
// paramétré : le domaine ne doit ni le choisir, ni le régler. L'implémentation
// vient de `internal/infrastructure/security` (Argon2id).
type HashSecret = func(plain string) (string, error)

// VerifySecret compare un secret à son condensé.
//
// L'implémentation DOIT comparer en temps constant. Une comparaison qui s'arrête
// au premier octet différent laisse mesurer combien de caractères sont corrects.
type VerifySecret = func(plain, encoded string) (bool, error)

// ─── Ports d'effets purs : l'horloge et l'aléa ───────────────────────────────

// Now retourne l'instant courant.
//
// Port pour que les tests soient déterministes : l'expiration d'une session se
// vérifie en avançant une variable, pas en attendant.
type Now = func() time.Time

// NewToken produit un jeton opaque.
//
// L'implémentation DOIT tirer d'une source cryptographiquement sûre. Un jeton
// prévisible est une authentification contournée, et `math/rand` en produit.
type NewToken = func() (domain.Token, error)

// NewIdentityID produit un identifiant d'identité.
type NewIdentityID = func() domain.IdentityID
