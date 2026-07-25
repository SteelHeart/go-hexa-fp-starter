# Où trouver quoi

## Dans ce dépôt

| Je cherche | C'est ici |
|---|---|
| Le règlement d'ingénierie | [`rules/`](README.md) — **fait foi** |
| Ce qui est absolument interdit | [`rules/interdictions.md`](interdictions.md) |
| La barre pour livrer | [`rules/definition-of-done.md`](definition-of-done.md) |
| Une décision d'architecture et son pourquoi | [`documentation/adr/`](../documentation/adr/README.md) — **fait foi** |
| Comment nommer une branche, un commit, un fichier | [`documentation/process/NOMENCLATURE.md`](../documentation/process/NOMENCLATURE.md) |
| Les labels et le board | [`documentation/process/LABELS.md`](../documentation/process/LABELS.md) |
| Les failles ouvertes | [`documentation/securite/registre-securite.md`](../documentation/securite/registre-securite.md) |
| Qui a le droit d'appeler quoi | [`documentation/securite/matrice-acces.md`](../documentation/securite/matrice-acces.md) |
| Ce qui manque au socle et fait perdre du temps | [`documentation/process/JOURNAL_FRICTION.md`](../documentation/process/JOURNAL_FRICTION.md) |
| Un exemple complet de feature | `internal/features/user_registration/` |
| Le contrat d'API | `api/openapi.yaml` — **généré**, jamais édité à la main |

## Hiérarchie des sources en cas de contradiction

1. **ADR** — tranche l'architecture. Gagne sur tout le reste.
2. **`rules/`** — règlement d'ingénierie.
3. **`CLAUDE.md`** — amorçage, résumé. Ne fait jamais foi contre un ADR ou une règle.
4. **README, commentaires de code** — indicatifs.

Une contradiction découverte se corrige **dans la même PR** que celle qui l'a révélée. Une
contradiction laissée en place rend tout le corpus non fiable, et donc inutilisé.

## Le socle est réutilisable tel quel

Ce dépôt ne dépend d'aucune personne ni d'aucune organisation :

- aucun pseudo, aucune équipe, aucun `CODEOWNERS` — les contraintes portent sur des **règles**,
  vérifiées par la CI ;
- le **chemin de module** est la seule valeur nominative, isolée derrière `task rename` ;
- la feature `user_registration` est un **exemple de référence**, à supprimer ou remplacer — elle
  n'est requise par rien d'autre que ses propres tests.

Pour démarrer un projet à partir de ce socle, voir [`toolchain.md`](toolchain.md) § 7.

## Vocabulaire imposé

Pour que les recherches dans le code et la documentation aboutissent, un concept porte **un seul**
nom :

| On écrit | On n'écrit pas |
|---|---|
| **feature** | module, bounded context, domaine, service |
| **port** | interface, contrat, repository, gateway |
| **adaptateur primaire** | contrôleur, handler, delivery, entrypoint |
| **adaptateur secondaire** | repository, infrastructure, DAO |
| **cas d'usage** | service, interactor, business logic |
| **décorateur** | middleware (réservé au HTTP), wrapper, intercepteur |
| **composition root** | container, DI, bootstrap, wiring |
