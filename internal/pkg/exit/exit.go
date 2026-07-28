// Package exit porte les codes de sortie d'un programme, selon `sysexits.h`.
//
// # Pourquoi une convention plutôt que 0 et 1
//
// Un binaire de ligne de commande est appelé par d'autres programmes bien plus
// souvent que par des humains : scripts de déploiement, `Makefile`, tâches
// planifiées, jobs de CI. Ces appelants ne lisent pas les messages ; ils lisent
// le code de retour, et ils en ont besoin pour décider s'il faut **réessayer**.
//
// Avec `1` pour tout, un mot de passe trop court et une base injoignable sont
// indiscernables : le script réessaie l'un — inutilement, à l'infini — ou
// abandonne sur l'autre, alors qu'une seconde tentative aurait suffi.
//
// # Pourquoi `sysexits.h` et pas une table maison
//
// Parce qu'elle existe depuis 1980, qu'elle est celle de `sendmail`, de `git` et
// de la plupart des outils Unix, et qu'un opérateur qui voit `78` sait déjà
// qu'il s'agit d'une erreur de configuration. Une table inventée oblige à lire
// notre documentation pour interpréter notre sortie.
//
// # Ce paquet est sans dépendance
//
// Des constantes entières, rien d'autre. C'est ce qui lui permet de vivre dans
// `internal/pkg/` et d'être utilisé par n'importe quel binaire sans rien tirer
// derrière lui.
package exit

// Codes de sortie, tels que définis par `sysexits.h` (BSD).
//
// Seuls ceux que ce socle sait produire sont déclarés. En ajouter « au cas où »
// laisserait croire qu'un chemin les rend, alors que rien ne les émet — la
// même faute que déclarer un pilote qui n'existe pas (ADR 014).
const (
	// OK : tout s'est bien passé.
	OK = 0

	// Usage : la ligne de commande est mal formée — option inconnue, argument
	// manquant. L'appelant doit corriger sa commande, jamais réessayer.
	Usage = 64

	// DataErr : les données fournies sont invalides. C'est la faute de
	// l'utilisateur, pas du service : réessayer à l'identique échouera pareil.
	DataErr = 65

	// Unavailable : un service dont on dépend est injoignable. C'est le SEUL
	// code de cette liste qui autorise un réessai — et c'est toute son utilité.
	Unavailable = 69

	// Software : une erreur interne. Réessayer ne coûte rien mais ne promet
	// rien ; ce qu'il faut, c'est lire les journaux.
	Software = 70

	// Config : la configuration est incohérente ou incomplète. Distinct de
	// `Usage` : la commande était correcte, c'est l'environnement qui ne l'est
	// pas — donc ce n'est pas à l'appelant de corriger sa ligne de commande.
	Config = 78

	// NoPerm : l'opération est refusée par une garde, et le refus est
	// DÉLIBÉRÉ. Un `seed` en production le rend : ce n'est ni une panne, ni une
	// erreur de saisie, c'est un « non » assumé.
	NoPerm = 77
)
