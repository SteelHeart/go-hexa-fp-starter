// Package domain porte le vocabulaire de la notification, sans dépendance.
//
// # Pourquoi ce module existe
//
// `user.registered.v1` est publié depuis la première tranche verticale, et
// **personne ne s'y abonne**. La chaîne s'arrête au relais : l'événement est
// durable, transporté, et sans effet. Un module qui produit un événement que
// rien ne consomme est un module dont la moitié asynchrone n'a jamais tourné —
// et ce dépôt a déjà mesuré ce que coûte du code jamais exécuté.
//
// # Ce que ce module NE sait pas
//
// Il ignore `user_registration`, l'inscription, et le mot « bienvenue ». Il
// envoie un message à un destinataire sur un canal. C'est le composition root
// qui relie un événement métier à un message — et cette ignorance est ce qui
// rend le module réutilisable par `billing` demain sans le toucher.
//
// # Ce paquet est PUR
//
// Ni horloge, ni I/O, ni gabarit. Le rendu d'un contenu appartient à l'appelant :
// un module noyau qui embarquerait un moteur de gabarits imposerait sa syntaxe à
// toutes les applications du socle.
package domain

import "errors"

// Erreurs sentinelles du module.
//
// `internal/core/**` retourne `error` et non `Result[T, domain.Error]` : la
// frontière est nette et vérifiable, et un module noyau est technique.
var (
	// ErrIncomplete refuse un message mal formé AVANT tout envoi.
	//
	// Le refus est en amont pour deux raisons : il ne coûte aucun appel réseau,
	// et il ne laisse pas une adresse vide atteindre un fournisseur, où elle
	// deviendrait un rejet facturé et journalisé chez un tiers.
	ErrIncomplete = errors.New("message incomplet")

	// ErrUnknownChannel refuse un canal que ce module ne sait pas servir.
	//
	// Deny par défaut : un canal inconnu ne se résout jamais en « le plus
	// proche ». Se rabattre sur le courriel parce que le SMS n'est pas livré
	// enverrait un code de vérification à la mauvaise adresse.
	ErrUnknownChannel = errors.New("canal de notification inconnu")

	// ErrUndeliverable signale un échec du fournisseur.
	//
	// Distinct d'`ErrIncomplete` : le message était valide, c'est l'envoi qui a
	// échoué. La différence décide du réessai — on rejoue une panne, jamais une
	// adresse invalide.
	ErrUndeliverable = errors.New("notification non délivrée")
)
