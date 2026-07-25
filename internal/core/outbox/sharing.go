package outbox

import (
	"errors"
	"fmt"
)

// ErrNotShared refuse un pilote qui ne franchit pas la frontière du processus.
//
// # Le défaut que ce refus empêche
//
// Le pilote `memory` vit DANS le processus. Un dépileur lancé comme binaire
// séparé interrogerait donc son propre magasin — vide — pendant que les
// événements écrits par le serveur resteraient dans la mémoire du serveur.
//
// Il tournerait sans rien publier ET sans aucune erreur : les journaux seraient
// propres, la sonde verte, le processus vivant. Le défaut ne se découvrirait que
// le jour où quelqu'un demande pourquoi un client n'a jamais reçu son courriel —
// et il faudrait alors remonter toute la chaîne pour trouver que le maillon
// tournait à vide depuis le premier jour.
//
// Un composant silencieusement inerte est le pire défaut possible : c'est le
// seul qui ne se signale jamais.
var ErrNotShared = errors.New("pilote d'outbox non partagé entre processus")

// SharedAcrossProcesses indique si un pilote est visible depuis un AUTRE
// processus que celui qui écrit.
//
// Cette connaissance vit ici, dans le module, et non chez l'appelant : le module
// est le seul à savoir ce que chacun de ses pilotes garantit. Un nouveau pilote
// ajoute donc sa réponse ici, au même endroit que son montage — plutôt que dans
// une liste tenue à jour ailleurs, qui finirait par diverger.
//
// Deny par défaut : un pilote inconnu est réputé NON partagé. Se tromper dans ce
// sens fait échouer un démarrage ; se tromper dans l'autre fait tourner un
// dépileur à vide.
func SharedAcrossProcesses(driver string) bool {
	switch driver {
	case driverPostgres:
		return true
	case driverMemory:
		return false
	default:
		return false
	}
}

// RequireSharedDriver refuse une configuration sur laquelle un dépileur SÉPARÉ
// ne verrait jamais rien.
//
// À appeler par tout binaire de dépilage, jamais par le serveur : celui-ci écrit
// dans l'outbox et n'a donc aucune raison d'exiger un pilote partagé.
func RequireSharedDriver(driver string) error {
	if SharedAcrossProcesses(driver) {
		return nil
	}
	return fmt.Errorf(
		"%w: %q. Un dépileur lancé séparément ne verrait jamais les événements "+
			"écrits par le serveur. Basculer modules.%s.driver sur un pilote partagé",
		ErrNotShared, driver, Name)
}
