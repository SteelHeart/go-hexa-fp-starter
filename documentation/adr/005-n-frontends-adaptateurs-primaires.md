# ADR 005 — N frontends simultanés comme adaptateurs primaires

- **Statut** : Accepté
- **Date** : 2026-07-25

## Contexte

Le socle doit servir **plusieurs surfaces à la fois** : application web, application mobile,
interface en ligne de commande, consommateurs d'événements — et d'autres non encore identifiées.

Le réflexe habituel est de faire du HTTP le chemin canonique, puis de faire appeler l'API par la
CLI et par les tâches planifiées. Cela paraît économe, et c'est le début de la dégradation : la CLI
hérite alors de l'authentification HTTP, de la sérialisation JSON et de la latence réseau pour
appeler du code qui tourne dans le même processus. Et le premier besoin qui ne rentre pas dans une
route REST se retrouve implémenté deux fois.

## Décision

**Un frontend est un adaptateur primaire. Rien d'autre.** Toutes les surfaces appellent la **même
fonction de cas d'usage**, en mémoire.

```
adapters/primary/http/    → web + mobile
adapters/primary/cli/     → terminal, scripts, CI
adapters/primary/events/  → consommateur asynchrone
adapters/primary/grpc/    → le jour où c'est utile
```

Invariant vérifiable : **ajouter une surface ne modifie aucun fichier de `domain/`, `ports/` ou
`application/`.** Une PR qui ajoute une surface et touche ces dossiers est refusée.

Corollaires :

- **Aucun DTO partagé entre surfaces.** Chaque adaptateur possède ses types d'entrée et de sortie.
  Un DTO commun fige web et mobile ensemble et rend toute évolution cassante pour les deux.
- Le découpage en **binaires** suit le cycle de vie opérationnel (`server`, `worker`, `cli`), pas
  le nombre de surfaces : un binaire peut monter plusieurs adaptateurs primaires.
- Le cœur ne connaît **aucun** mécanisme d'authentification. L'identité entre comme paramètre de
  commande, déjà établie par la surface.
- Les clients (TypeScript, Dart, Swift) sont **générés** depuis `api/openapi.yaml`, lui-même
  généré depuis le code.

## Conséquences

### Ce que ça achète

- Ajouter une surface est une tâche locale, chiffrable, sans risque de régression métier.
- La CLI et les tâches planifiées appellent le métier **sans réseau ni sérialisation**.
- Chaque surface évolue à son rythme : le mobile peut rester sur `/v1` pendant que le web passe
  sur `/v2`.

### Ce que ça coûte

- Des présentateurs par surface — duplication apparente, délibérée.
- Chaque surface doit traiter **exhaustivement** les `ErrorCode`. Ajouter un code d'erreur impose
  de passer dans toutes les surfaces (le linter `exhaustive` le rend obligatoire).
- Plus de code de traduction que dans une architecture mono-frontend.

### Ce que ça rend impossible

- Faire appeler l'API HTTP par la CLI.
- Partager un type de réponse entre web et mobile.
- Mettre une règle métier dans une surface.

## Alternatives écartées

| Alternative | Pourquoi écartée |
|---|---|
| HTTP canonique, les autres surfaces appellent l'API | Réseau et sérialisation pour du code en processus ; duplication dès le premier besoin non-REST |
| Un BFF par surface, au-dessus d'une API interne | Une couche de plus à déployer et à versionner ; justifié seulement avec des équipes séparées par surface |
| gRPC/Connect comme transport unique | Bon pour web + mobile + service-à-service, mais impose protobuf comme source de vérité ; reste ajoutable comme adaptateur supplémentaire sans rien casser |

## Garde

`arch-go` (une surface ne peut pas importer `application/`), `exhaustive` (traduction complète des
codes d'erreur), et la case « ajouter la surface n'a modifié aucun fichier du cœur » du gabarit de PR.
