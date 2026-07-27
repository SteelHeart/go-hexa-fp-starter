// Package security fournit le hachage de mot de passe et le chiffrement au repos.
//
// Rien ici ne connaît le métier : ce sont des primitives, branchées sur des
// ports par le composition root.
//
// # Un fichier par responsabilité publique
//
// Trois primitives indépendantes, trois fichiers (rules/tests.md §2) :
//
//	hasher.go       Argon2id — hacher, vérifier, décider du réhachage
//	cipher.go       AES-256-GCM — chiffrer et déchiffrer au repos
//	blind_index.go  index déterministe pour chercher sans déchiffrer
//
// Le découpage n'est pas cosmétique ici : chacun de ces trois fichiers porte au
// moins une constante dont la valeur EST une garantie de sécurité — la longueur
// de clé AES, les bornes du condensé décodé. Une garantie noyée au milieu de
// deux cents lignes d'un autre sujet se relit mal, et se modifie sans qu'on
// mesure ce qu'on relâche.
package security
