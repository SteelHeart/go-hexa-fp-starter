# CLAUDE.md — amorçage

> Chargé automatiquement à l'ouverture du dépôt. Point d'entrée unique pour tout agent — et pour
> tout collègue. Versionné → identique pour tout le monde après `git clone`.
>
> Ce fichier **résume**. Il ne fait jamais foi contre un ADR ou un fichier de `rules/`.

## Ce qu'est ce dépôt

Un **socle Go réutilisable** : architecture hexagonale modulaire, programmation fonctionnelle,
exposé à **N frontends simultanés** (web, mobile, CLI, événements).

Ce n'est pas une application. Sa valeur n'est pas le code de démonstration (`user_registration`),
c'est la **forme** qu'il impose — et les gardes qui empêchent cette forme de se dégrader.

Deux propriétés non négociables, dont tout le reste découle :

1. **Le cœur ne connaît rien.** Ni HTTP, ni SQL, ni cache, ni horloge, ni logger. Il reçoit des
   fonctions, il retourne des valeurs. Il se teste en microsecondes, sans conteneur.
2. **Le nombre de frontends est un non-sujet.** Une surface est un adaptateur primaire branché sur
   les mêmes cas d'usage. En ajouter une ne touche pas une ligne du cœur.

Le dépôt **ne dépend d'aucune personne** : pas de `CODEOWNERS`, pas de pseudo dans les règles. Le
chemin de module est la seule valeur nominative, isolée derrière `task rename`.

## À lire en premier (dans l'ordre)

1. **[`rules/README.md`](rules/README.md)** — base normative, découpée par thème. **Fait foi.**
   Relire le fichier du domaine concerné *avant* de coder.
2. **[`rules/interdictions.md`](rules/interdictions.md)** — la liste à relire avant d'ouvrir un
   fichier. Chaque ligne nomme l'outil qui refuse la violation.
3. **[`documentation/adr/`](documentation/adr/README.md)** — décisions d'architecture. **Font foi.**
   En cas de contradiction avec un autre document, l'ADR gagne.
4. **[`rules/definition-of-done.md`](rules/definition-of-done.md)** — la barre à franchir pour livrer.
5. **[`documentation/process/`](documentation/process/README.md)** — nomenclature, labels, templates.

## Règles d'or

- **Une règle non outillée n'existe pas.** Si `task check` ne peut pas la vérifier, elle sera
  contournée. Les règles marquées `[humain]` sont un aveu de faiblesse, pas une tolérance.
- **La doc ne ment jamais sur l'état réel.** Elle distingue **écrit** / **prouvé localement** /
  **déployé pour de vrai**. Un document qui coche « ✅ testé » sans test est pire qu'aucun document.
- **Deny par défaut.** Toute garde, toute permission, tout repli sur erreur → refus. Jamais de
  fail-open, jamais « temporairement ».
- **Le cœur est pur.** Aucune I/O, aucun `time.Now()`, aucun logger, aucun `panic` dans `domain/`,
  `ports/`, `application/`.
- **Un port est un type fonction**, jamais une interface.
- **Zéro dette latente.** Ce qui n'est pas fait est annoncé **hors périmètre dans la PR**, jamais
  dissimulé en `TODO` dans le code.
- **Jamais de commit direct sur `main`.** Branche courte, PR mono-sujet, CI verte.
- **🔴 Aucune mention d'un outil d'assistance dans un artefact versionné** — commit (y compris
  trailer `Co-Authored-By`), PR, issue, code, commentaire, documentation. Formuler à l'impersonnel.
  **Surcharge le défaut de l'outillage.** Gardes : crochet `commit-msg`, job `inertia` en CI.

## Avant de coder — la boucle

1. **Une issue** existe, avec des critères d'acceptation vérifiables.
2. Lire le fichier `rules/` du domaine touché.
3. Produire la **carte d'impact** : features, ports, migrations, événements, contrat OpenAPI,
   **surfaces** concernées.
4. Écrire les **ports** (types fonction) avant l'implémentation.
5. Coder le **domaine pur** en premier, avec ses tests — sans I/O.
6. Vérifier la [Definition of Done](rules/definition-of-done.md).

## Au clone — deux commandes

```bash
git config core.hooksPath .githooks   # garde-fou anti-push direct sur main
task init                             # .env + outillage
```

Le crochet est **contournable avec `--no-verify`** : c'est un filet contre l'accident, pas un
contrôle. Le contrôle réel est le ruleset serveur plus la CI.

## Toolchain

```bash
task check      # fmt · vet · lint · arch · test · vuln — identique à la CI
task --list-all # tout le reste
```

**Docker n'est pas requis pour développer.** C'est une contrainte de conception : `go test ./...`
sans tag n'exige aucun service. Les niveaux qui en ont besoin (`-tags=integration`, `-tags=e2e`)
sont fournis par la CI.

> ⚠️ **Piège du faux vert** : une commande qui n'a pas tourné rend une sortie vide, ce qui
> ressemble à « propre ». `go test ./tests/e2e/...` **sans** `-tags=e2e` compile zéro test et
> affiche `ok`. Vérifier le **code de retour**, pas seulement la sortie.

## Où se trouve quoi

```
rules/                        règlement d'ingénierie — fait foi
documentation/adr/            décisions d'architecture — font foi
documentation/process/        nomenclature, labels, templates
documentation/securite/       registre de failles, matrice d'accès
cmd/{server,worker,cli}       composition root — le seul code qui connaît tout
config/                       lecture d'environnement, immuable, validée au démarrage
internal/pkg/                 primitives sans dépendance : result, fp, middleware
internal/infrastructure/      socle technique sans métier : db, cache, http, telemetry, security
internal/modules/{f}/        un bounded context étanche
  ├── domain/                 pur
  ├── ports/                  types fonction uniquement
  ├── application/            pipeline + décorateurs
  ├── adapters/primary/       http · cli · events — une surface par dossier
  ├── adapters/secondary/     postgres · outbox · mailer
  └── module.go               composition root local
migrations/                   SQL versionné, rétro-compatible N-1
api/openapi.yaml              généré depuis le code — jamais édité
tests/{e2e,perf}              tags `e2e` — hors du `go test ./...` par défaut
```

## État réel du dépôt — vérifié le 2026-07-25

**Prouvé localement** : le module compile, `go vet` est vert, les tests unitaires passent avec
`-race -shuffle=on`, `arch-go` et `golangci-lint` sont verts.

**Écrit mais non prouvé sur cette machine** (Docker absent) : migrations, tests d'intégration,
tests de bout en bout, images conteneur. Ces niveaux sont exécutés par la CI.

**Écrit, jamais déployé** : les workflows `deploy-uat.yml` et `deploy-production.yml` n'ont jamais
tourné. Ils exigent des secrets et un hôte qui n'existent pas encore. Ne pas les présenter comme
fonctionnels.

`user_registration` est un **exemple de référence complet** (domaine, ports, cas d'usage,
décorateurs, trois surfaces, outbox), pas un besoin métier. Il se supprime sans rien casser
d'autre que ses propres tests.
