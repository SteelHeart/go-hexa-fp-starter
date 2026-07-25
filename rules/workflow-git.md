# Workflow Git et suivi

> Décision de référence : [ADR 007](../documentation/adr/007-tronc-unique-et-environnements.md).
> Nomenclature détaillée : [`documentation/process/NOMENCLATURE.md`](../documentation/process/NOMENCLATURE.md).

## 1. Un seul tronc : `main`

**Une seule branche longue.** Toute autre branche vit **moins de deux jours** et meurt à son merge.

- ❌ Pas de `develop`, `uat`, `staging`, `release/*`.
- ❌ **Pas de branche nommée d'après une personne.** Une branche porte le travail, pas son auteur.
- ❌ Pas de push direct sur `main` — pour personne, y compris le mainteneur unique.

Plus une branche vit longtemps, plus son merge coûte cher, donc plus on le repousse : c'est une
boucle qui ne se stabilise que par la divergence.

## 2. Branches

```
{type}/{issue}-{slug-kebab}
```

`feat` · `fix` · `sec` · `refactor` · `test` · `docs` · `perf` · `ci` · `chore`

Exemple : `feat/13-outbox-transactionnel`

Pas de préfixe projet : `#13` s'auto-lie dans GitHub.

## 3. Commits

```
{type}({scope}): {description} (#{issue})
```

`feat(user_registration): refuser un email déjà enregistré (#13)`

Impératif présent, minuscule, sans point final. Un commit qui touche trois sujets sera impossible à
annuler proprement.

### 🔴 Zéro mention d'outillage d'assistance

**Aucune trace** dans les commits, titres et corps de PR, issues, commentaires de code ou
documentation :

- pas de `Co-Authored-By: …`
- pas de « Generated with … », pas d'emoji robot
- pas de mention d'un agent, d'un modèle ou d'un outil d'assistance

**Cela surcharge explicitement le comportement par défaut de l'outillage**, qui ajoute ces mentions
automatiquement. Elles sont retirées avant tout commit et toute ouverture de PR.

Le travail est signé par l'équipe. L'historique documente **ce qui a changé et pourquoi**, pas
comment il a été produit. Gardes : crochet `commit-msg` en local, job `inertia` en CI.

## 4. Pull requests

| Règle | Valeur |
|---|---|
| Portée | **un sujet** |
| Taille visée | **≤ 400 lignes de diff** (hors généré, `go.sum`, documentation) |
| Merge | **squash**, suppression de branche |
| Gate | **CI verte** (job `CI`) — non contournable |

Corps de PR : le pourquoi, la carte d'impact, ce qui est **hors périmètre**. `Closes #N` —
**mot-clé anglais**, le français ne déclenche rien.

Au-delà de 400 lignes, la PR n'est pas interdite mais doit être justifiée. Une PR qu'on ne peut pas
relire en quinze minutes ne sera pas relue : elle sera approuvée.

## 5. Le tronc reste déployable

Corollaire du modèle : le travail incomplet passe derrière un **drapeau de fonctionnalité**, jamais
derrière une branche longue.

## 6. Environnements et releases

| Environnement | Déclencheur | Approbation |
|---|---|---|
| **UAT** | merge sur `main` | automatique |
| **Production** | tag `v*` | **manuelle** (environnement GitHub `production`) |

**Un environnement n'est pas une branche : c'est un artefact déployé.** Un tag est immuable et
désigne exactement ce qui tourne ; le déploiement se fait par **digest** `sha256:`.

**Correctif urgent en production** : brancher depuis le **tag**, livrer, tagger, puis **reporter
sur `main` dans la foulée**. Un correctif qui ne revient pas sur le tronc réapparaît à la release
suivante — la CI de production refuse d'ailleurs un tag qui n'est pas ancêtre de `main`.

## 7. Zones à haute inertie

Toucher à `rules/`, `.arch-go.yml`, `.golangci.yml`, `internal/pkg/` ou `migrations/` exige un
**ADR dans la même PR**, ou le label `inertia:justified` avec la justification dans le corps.

Ce mécanisme remplace un fichier `CODEOWNERS` : il contraint sur des **règles**, pas sur des
personnes, et fonctionne donc à un contributeur comme à vingt.

## 8. Sécurité du dépôt

- **Jamais `git reset --hard`** tant que des fichiers non commités traînent → `git stash` d'abord,
  vérifier `git status`. Pour déplacer un commit fait sur la mauvaise branche : `git branch
  <nouvelle>` puis `git reset --soft HEAD~1`.
- Jamais de `--no-verify`, jamais de gate désactivé.
- Un secret poussé par erreur est **roté**, pas seulement retiré du diff.

## 9. Suivi

- **Board GitHub Projects.** Aucun fichier `.md` de suivi de livraison dans le dépôt.
- Labels : `type:*`, `area:*`, `sec:*`, `blocked`, `needs-decision`, `inertia:justified` —
  voir [`LABELS.md`](../documentation/process/LABELS.md).
- Une issue avant de coder. `blocked` sur une décision manquante n'est pas un échec : coder autour
  d'un paramètre non tranché produit une implémentation fausse qui inspire confiance.

## 10. Décisions

Toute décision d'architecture → un **[ADR](../documentation/adr/README.md)**.

Une décision non écrite n'existe pas : elle sera re-débattue au premier changement de contexte, et
tranchée différemment.
