<!--
Le TITRE de cette PR devient le message du squash commit sur le tronc.
Il doit respecter Conventional Commits — la CI le vérifie (job `pr-title`) :
  {type}({scope}): {description}
  ex. feat(user_registration): refuser un email déjà enregistré

🔴 Aucune mention d'outillage d'assistance ici ni dans les commits.
   Formuler à l'impersonnel. Vérifié par la CI (job `inertia`).
-->

Closes #

## Pourquoi

<!-- Le problème, pas la solution. -->

## Ce que fait cette PR

-
-

## Hors périmètre

<!--
Ce qui n'est PAS fait, et qui aurait pu l'être. Obligatoire : zéro dette latente.
Un contournement ne supprime pas le problème, il supprime le signal.
Écrire « rien » si la PR est complète.
-->

## Carte d'impact

| Axe | Impact |
|---|---|
| Features touchées | |
| Ports ajoutés / modifiés | |
| Migration | non / oui — rétro-compatible N-1 ? |
| Événement outbox | non / oui — consommateur idempotent ? |
| Contrat OpenAPI | non / oui — `task openapi` relancé ? |
| Surfaces | web · mobile · CLI · événements · aucune |
| Zone à haute inertie | non / oui — ADR joint ou label `inertia:justified` |

## Definition of Done

<!-- rules/definition-of-done.md — décocher ce qui ne s'applique pas, ne rien laisser de faux. -->

- [ ] Règles métier en **fonctions pures** dans `domain/`, effets injectés
- [ ] `domain/`, `ports/`, `application/` sans transport, persistance ni logger
- [ ] Aucune feature n'importe une autre feature
- [ ] Nouveaux ports = **types fonction**, avec contrat d'erreur documenté
- [ ] Chaque nouveau `ErrorCode` est atteint par un test
- [ ] Un bug corrigé porte son test de non-régression
- [ ] Cliquets tenus : 70 % global, 90 % sur `domain/` + `application/`
- [ ] Aucun secret, aucune donnée personnelle dans les logs
- [ ] Ajouter une surface n'a modifié aucun fichier du cœur
- [ ] La doc reflète l'**état réel** (écrit / prouvé localement / déployé)
- [ ] `task check` vert en local — **code de retour vérifié**

## Comment vérifier à la main

```bash
task check
# puis, si la pile est disponible :
task up && task run
curl -i -X POST http://localhost:8080/v1/users \
  -H 'Content-Type: application/json' \
  -d '{"email":"a@b.co","password":"correct-horse-battery-staple"}'
```
