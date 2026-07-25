# Definition of Done

La barre à franchir pour dire « livré ». Tant qu'une ligne n'est pas cochée, le travail n'est pas
terminée — elle est *en cours*. Cette liste est reprise telle quelle dans le gabarit de PR.

## 1. Besoin

- [ ] Une **issue** existe, avec des critères d'acceptation explicites et vérifiables.
- [ ] La **carte d'impact** est produite : modules touchés, ports ajoutés ou modifiés, migrations,
      événements outbox, contrat OpenAPI, **surfaces concernées** (web / mobile / CLI / événements).
- [ ] Ce qui est **hors périmètre** est écrit dans la PR — pas laissé en `TODO` dans le code.

## 2. Contrat

- [ ] Les nouveaux ports sont des **types fonction** dans `ports/`, avec leur **contrat d'erreur**
      documenté au-dessus.
- [ ] Aucun identifiant en `string` nue ni montant en `float` dans une signature de domaine.
- [ ] `api/openapi.yaml` régénéré (`task openapi`) si le contrat HTTP a changé.
- [ ] Un changement cassant d'API porte une **nouvelle version de route**, il ne modifie pas la
      version existante.

## 3. Code

- [ ] Les règles métier sont des **fonctions pures** dans `domain/` ; les effets sont injectés.
- [ ] `domain/`, `ports/`, `application/` n'importent ni transport, ni persistance, ni logger.
- [ ] Aucun module métier n'importe un autre module métier.
- [ ] Les préoccupations transverses sont des **décorateurs**, pas du code inséré dans le cas d'usage.
- [ ] Aucun secret, aucune donnée personnelle dans les logs.
- [ ] `arch-go` vert · `golangci-lint` vert · `go vet` vert.

## 4. Sécurité

- [ ] **Deny par défaut** vérifié sur les nouveaux chemins, avec un test qui prouve le refus.
- [ ] Ligne(s) ajoutée(s) à la **matrice rôle × endpoint** pour toute nouvelle route.
- [ ] Si le module touche l'argent ou la preuve : **idempotence** et journal d'audit vérifiés.
- [ ] `govulncheck` et `gitleaks` verts.

## 5. Données

- [ ] La migration est **rétro-compatible avec la version N-1** du code.
- [ ] Le `-- +goose Down` a été **réellement exécuté**, pas seulement écrit.
- [ ] Toute publication d'événement passe par l'**outbox transactionnel**.
- [ ] Tout consommateur d'événement est **idempotent**, et ça se voit dans un test.

## 6. Tests

- [ ] Domaine : cas nominaux, cas limites, entrées invalides.
- [ ] Cas d'usage : chemin nominal **et** chaque chemin d'erreur.
- [ ] Chaque nouveau `ErrorCode` est atteint par au moins un test.
- [ ] Une nouvelle implémentation de port est inscrite dans la **suite de conformité**.
- [ ] Un bug corrigé porte son **test de non-régression**, écrit avant le correctif.
- [ ] Bout en bout sur le parcours critique de **chaque surface** touchée.
- [ ] Cliquets tenus : **70 %** global, **90 %** sur `domain/` + `application/`.
- [ ] Suite verte avec `-race -shuffle=on`, **code de retour vérifié** (pas de faux vert).

## 7. Surfaces

- [ ] Chaque surface concernée traite **exhaustivement** les `ErrorCode` (le linter `exhaustive` y veille).
- [ ] Aucun DTO partagé entre deux surfaces.
- [ ] Ajouter la surface n'a modifié **aucun** fichier de `domain/`, `ports/` ou `application/`.

## 8. Documentation

- [ ] **ADR écrit** si une décision d'architecture a été prise, ou si la PR touche une zone à haute
      inertie (`rules/`, `.arch-go.yml`, `.golangci.yml`, `internal/pkg/`, `migrations/`).
- [ ] La doc reflète l'**état réel** — ce qui est bouchonné est écrit « bouchon » ; ce qui est
      **écrit** n'est pas présenté comme **prouvé** ni comme **déployé**.
- [ ] Le fichier remplacé est marqué obsolète en tête, avec sa date et son remplaçant.

## 9. Revue

- [ ] PR mono-sujet, ≤ 400 lignes de diff, **CI verte**.
- [ ] **Aucune mention d'outillage d'assistance** dans les commits, le titre et le corps de la PR.
- [ ] Le titre de PR respecte Conventional Commits — il devient le message du squash commit.
