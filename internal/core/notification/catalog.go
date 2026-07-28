package notification

import "github.com/SteelHeart/go-hexa-fp-starter/internal/config"

// Catalog déclare les pilotes de ce module — ADR 014.
//
// Aucun fichier du framework ne nomme `notification` : le catalogue est
// construit ici, à côté de la fabrique `New`, et il PARTAGE ses constantes avec
// elle. La divergence entre ce qui est déclarable et ce qui est constructible
// devient donc impossible.
//
// # Un seul pilote, et il n'envoie rien
//
// `smtp`, `mailjet`, `ses` sont décrits dans
// documentation/technique/modules-noyau.md et ne sont PAS écrits. Ce dépôt a
// déjà payé la faute inverse : huit paquets de pilotes ont vécu des mois avec
// zéro test, à aucun niveau (#37), et deux défauts de production y dormaient. Un
// pilote SMTP écrit sans serveur pour l'éprouver est du code qui a l'air de
// marcher — et sur ce module, « a l'air de marcher » veut dire que personne ne
// reçoit rien pendant des semaines.
func Catalog() config.ModuleCatalog {
	return config.ModuleCatalog{
		Name: {
			// Le défaut n'exige RIEN : ni serveur SMTP, ni compte chez un
			// fournisseur. C'est ce qui permet à la chaîne complète —
			// inscription, outbox, relais, notification — de partir d'un
			// `go run`.
			Default: driverLog,
			Drivers: map[string]config.Resources{
				// N'envoie à personne : voir les NON-garanties du paquet
				// drivers/log avant de l'envisager ailleurs qu'en
				// développement.
				driverLog: {Options: []string{OptionBody}},
			},
		},
	}
}
