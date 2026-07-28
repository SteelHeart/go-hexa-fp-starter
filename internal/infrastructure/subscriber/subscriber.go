// Package subscriber branche un consommateur d'événements sur les garanties du
// noyau.
//
// # Pourquoi ce paquet existe, et pourquoi ICI
//
// Il est le pendant exact de `relay` : celui-ci relie le dépileur au transport,
// celui-là relie le transport aux modules qui réagissent. Aucun des deux ne peut
// vivre dans un module noyau — `idempotency` ne doit importer aucune
// infrastructure, sous peine de ne plus pouvoir être extraite en module Go
// indépendant (ADR 012) — ni dans `cmd/`, parce que du code dans `main` n'est
// testable qu'à moitié.
//
// # Ce que ce paquet garde, et que personne ne remarque quand ça marche
//
// Trois choses que tout consommateur doit faire et que tout le monde oublie :
//
//  1. **Ne pas rejouer un effet.** Tous les transports d'ici sont « au moins une
//     fois ». Un courriel de bienvenue envoyé deux fois est le symptôme visible ;
//     un débit rejoué est le symptôme coûteux.
//  2. **Restaurer la trace.** Sans elle, la moitié asynchrone d'une requête
//     apparaît comme une trace orpheline, et personne ne relie jamais l'échec
//     d'envoi à l'inscription qui l'a causé.
//  3. **Ne pas confondre « déjà fait » et « échec ».** Le premier acquitte, le
//     second rejoue.
//
// # Carte des fichiers
//
//	subscriber.go   la carte du paquet et les erreurs
//	once.go         l'effet n'a lieu qu'UNE fois, même sur rejeu
//	trace.go        le contexte de trace repris de l'enveloppe
package subscriber

import "errors"

// ErrMissingID refuse une enveloppe sans identifiant.
//
// L'identifiant EST la clé d'idempotence : sans lui, aucun rejeu ne peut être
// reconnu, et le décorateur laisserait passer chaque copie en croyant bien
// faire. Un refus bruyant vaut mieux qu'une garantie silencieusement absente.
var ErrMissingID = errors.New("enveloppe sans identifiant : idempotence impossible")
