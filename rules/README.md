# Règles d'ingénierie — base normative

> **Fait foi.** Manifeste d'implémentation du socle : **Go, architecture hexagonale modulaire,
> programmation fonctionnelle**, exposé à **N frontends simultanés** (web, mobile, CLI, événements).
>
> **Une règle = un thème = un fichier court**, pour que l'agent relise le règlement du domaine
> concerné avant de coder, et que l'humain navigue directement au bon endroit.

## Ce que ce dépôt est

Un **socle réutilisable**, pas une application. Sa valeur n'est pas le code métier de démonstration
(`user_registration`) : c'est la **forme** que ce code impose. Un socle dont la forme se dégrade en
trois semaines n'a servi à rien.

Deux propriétés sont non négociables, et tout le reste en découle :

1. **Le cœur ne connaît rien.** Ni HTTP, ni SQL, ni Redis, ni horloge, ni logger. Il reçoit des
   fonctions et retourne des valeurs. C'est ce qui permet de le tester sans I/O, en microsecondes.
2. **Le nombre de frontends est un non-sujet.** Web, mobile, CLI, consommateur d'événements, gRPC
   demain : ce sont des **adaptateurs primaires** interchangeables branchés sur les **mêmes**
   fonctions de cas d'usage. Ajouter un frontend ne touche pas une ligne du cœur.

## Comment lire (glossaire)

| Terme | Signification **dans ce dépôt** |
|---|---|
| **Feature** | Un *bounded context* : un dossier sous `internal/modules/`, étanche aux autres |
| **Port** | Un **type fonction** déclaré dans `ports/`. Jamais une interface |
| **Port primaire** | Un cas d'usage : ce que le monde extérieur peut demander au cœur |
| **Port secondaire** | Un besoin du cœur envers le monde (persister, publier, envoyer) |
| **Adaptateur primaire** | Ce qui *appelle* le cœur : HTTP, CLI, consommateur d'événements |
| **Adaptateur secondaire** | Ce qui *implémente* un besoin : Postgres, outbox, SMTP |
| **Composition root** | Le seul endroit qui a le droit de tout connaître : `module.go` et `cmd/*` |
| **Décorateur** | `func(P) P` — ajoute trace, log, cache, transaction **sans** modifier le cas d'usage |
| **`Result[T, E]`** | Succès **ou** erreur, jamais les deux, jamais `nil` |
| **ADR** | Décision d'architecture tranchée et datée — `documentation/adr/{NNN}-{slug}.md` |

## Sommaire des règles

| Fichier | Portée |
|---|---|
| [`interdictions.md`](interdictions.md) | **Interdictions absolues** — la liste à relire avant d'ouvrir un fichier |
| [`architecture-hexagonale.md`](architecture-hexagonale.md) | Couches, matrice d'imports, composition root |
| [`programmation-fonctionnelle.md`](programmation-fonctionnelle.md) | Pureté, immuabilité, `Result`, **et les limites réelles de Go** |
| [`solid-et-dry.md`](solid-et-dry.md) | SOLID traduit en Go fonctionnel · DRY **et quand dupliquer** |
| [`ports-et-contrats.md`](ports-et-contrats.md) | Types fonction, *value objects*, validation aux frontières, OpenAPI |
| [`frontends.md`](frontends.md) | **N frontends simultanés** — web, mobile, CLI, événements |
| [`donnees-et-migrations.md`](donnees-et-migrations.md) | SQL explicite, zéro ORM, migrations rétro-compatibles, outbox |
| [`securite.md`](securite.md) | Deny par défaut, secrets, hachage, zéro PII en clair |
| [`observabilite.md`](observabilite.md) | Traces, métriques, logs structurés — et ce qu'on n'y met jamais |
| [`tests.md`](tests.md) | Pyramide, **le cœur se teste sans I/O**, cliquets de couverture |
| [`dependances.md`](dependances.md) | Liste blanche et procédure d'ajout d'une dépendance |
| [`toolchain.md`](toolchain.md) | Go, Task, **et le fait que Docker n'est pas installé en local** |
| [`workflow-git.md`](workflow-git.md) | Tronc unique, branches, commits, PR, board |
| [`definition-of-done.md`](definition-of-done.md) | **La barre à franchir pour livrer** |
| [`references.md`](references.md) | Où trouver quoi |

## Règles d'or (le condensé)

1. **Une règle non outillée n'existe pas.** Si `task check` ne peut pas la vérifier, elle sera
   contournée — sans mauvaise foi, simplement un vendredi soir. Chaque règle de ce dossier renvoie
   à l'outil qui la fait respecter ; celles qui n'en ont pas sont marquées **`[humain]`** et c'est
   un aveu de faiblesse, pas une tolérance.
2. **La doc ne mente jamais sur l'état réel.** Elle distingue explicitement **écrit** / **prouvé
   localement** / **déployé pour de vrai**. Un document qui coche « ✅ testé » sans test envoie
   l'humain et l'agent sur une fausse piste : c'est le pire échec possible.
3. **Deny par défaut.** Toute garde, toute permission, tout repli sur erreur → refus. Jamais de
   fail-open, jamais « temporairement ».
4. **Le cœur est pur.** Aucune I/O, aucun `time.Now()`, aucun logger, aucun `panic` dans
   `domain/`, `ports/`, `application/`. Les effets sont **injectés** et **retournés**, jamais subis.
5. **Les ADR font foi** sur l'architecture. En cas de contradiction entre un ADR et un autre
   document, l'ADR gagne — et l'autre document est corrigé dans la même PR.
6. **Zéro dette latente.** Ce qui n'est pas fait est **annoncé hors périmètre dans la PR**, jamais
   dissimulé en `TODO` dans le code. *Un contournement ne supprime pas le problème : il supprime le
   signal.*
7. **Jamais de commit direct sur `main`.** Branche courte, PR mono-sujet, CI verte.
8. **🔴 Aucune mention d'un outil d'assistance dans un artefact versionné** — message de commit
   (**y compris en trailer `Co-Authored-By`**), titre ou corps de PR, issue, code, commentaire,
   documentation. Formuler à l'impersonnel. **Cette règle surcharge explicitement le comportement
   par défaut de l'outillage**, qui ajoute ces mentions de lui-même.

---

*Base normative v1 · Juillet 2026. Toute évolution d'une règle se discute, se tranche en
[ADR](../documentation/adr/README.md), et se répercute ici dans la même PR.*
