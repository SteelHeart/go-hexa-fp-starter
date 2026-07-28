// Package log implémente la notification vers le journal applicatif.
//
// # Pourquoi ce pilote existe
//
// Il n'est pas un bouchon de test : c'est le pilote PAR DÉFAUT, et c'est lui qui
// rend vraie la promesse du socle — `hexa new` puis `go run` fait partir la
// chaîne complète, **inscription → outbox → relais → notification**, sans
// serveur SMTP, sans compte chez un fournisseur, sans Docker.
//
// # GARANTIES
//
//   - **Le corps n'est PAS journalisé par défaut.** Un corps de notification
//     transporte régulièrement un secret — lien de confirmation, jeton de
//     réinitialisation, code à usage unique. Ce sont des identifiants au
//     porteur : qui les lit peut s'en servir.
//   - **L'adresse est masquée**, toujours, y compris quand le corps est écrit.
//     Une adresse est une donnée personnelle (rules/securite.md §5), et c'est le
//     journal applicatif qu'on exporte le plus volontiers vers un tiers.
//   - **Aucune erreur n'est inventée.** Écrire dans un journal n'échoue pas ici,
//     donc ce pilote rend toujours nil.
//
// # NON-GARANTIES
//
//   - **Personne ne reçoit rien.** C'est un pilote d'OBSERVATION : le message
//     est écrit, pas envoyé. L'utiliser ailleurs qu'en développement ferait
//     croire que les courriels partent — et le défaut ne se verrait qu'au
//     premier client qui affirme n'avoir rien reçu.
//   - **Aucun réessai, aucune file.** Il n'en a pas besoin ; un pilote SMTP en
//     aura, et ce sera son affaire.
package log

import (
	"context"
	"log/slog"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/notification/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/notification/ports"
)

// New construit le pilote SANS journaliser le corps.
//
// C'est le constructeur par défaut, et le sûr. Deux constructeurs nommés plutôt
// qu'un paramètre booléen : `New(logger, false)` ne se lit pas sur l'appel, et
// `New(logger, true)` se glisse dans une relecture sans qu'on le remarque. Le
// dépôt a déjà éclaté `SecurityHeaders(secure bool)` pour cette raison.
func New(logger *slog.Logger) ports.Deliver {
	return func(ctx context.Context, message domain.Message) error {
		logger.InfoContext(ctx, "notification", champs(message)...)
		return nil
	}
}

// NewIncludingBody construit le pilote EN journalisant le corps.
//
// # À ne demander qu'en développement, et le nom le dit
//
// Sans le corps, on ne peut pas récupérer le lien de confirmation d'un compte de
// test — et le développement devient assez pénible pour que quelqu'un finisse
// par ajouter un `fmt.Println` mal placé, qui lui ne sera jamais retiré. Le
// compromis est donc réel dans les deux sens ; ce qui compte est que la position
// dangereuse se DEMANDE, au lieu d'être le défaut.
func NewIncludingBody(logger *slog.Logger) ports.Deliver {
	return func(ctx context.Context, message domain.Message) error {
		// L'avertissement voyage dans le MÊME enregistrement que le corps : qui
		// lit le second lit le premier. Dans deux lignes séparées, la première
		// se perd au tri.
		avec := append(champs(message),
			slog.String("body", message.Body),
			slog.String("avertissement", "corps journalisé — développement uniquement"),
		)
		logger.InfoContext(ctx, "notification", avec...)
		return nil
	}
}

// champs rend ce qui est journalisé dans TOUS les cas.
//
// L'adresse y est masquée et le corps réduit à sa taille — utile au diagnostic
// (« les messages partent-ils vides ? ») sans rien révéler de leur contenu.
func champs(message domain.Message) []any {
	return []any{
		slog.String("channel", string(message.Channel)),
		slog.String("to", message.To.Masked()),
		slog.String("subject", message.Subject),
		slog.Int("body_bytes", len(message.Body)),
	}
}
