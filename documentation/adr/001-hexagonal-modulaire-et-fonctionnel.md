# ADR 001 — Hexagonal modulaire et programmation fonctionnelle

- **Statut** : Accepté
- **Date** : 2026-07-25

## Contexte

Un socle destiné à porter plusieurs projets, sur plusieurs années, avec des frontends multiples et
des équipes qui changent. Deux modes de dégradation dominent dans ce type de base :

1. **La fuite technique vers le cœur.** Un tag ORM dans une entité, un `*http.Request` dans un
   service, un `time.Now()` dans une règle. Chaque occurrence est anodine ; leur accumulation rend
   le métier intestable sans infrastructure, donc mal testé, donc figé.
2. **Le couplage entre domaines.** Un import « juste pour ce type », une jointure « juste pour ce
   rapport ». Trois mois plus tard, plus rien ne s'extrait.

Ces deux dégradations ne se corrigent pas par de la discipline : elles se corrigent par des gardes.

## Décision

Le socle adopte l'**architecture hexagonale** (ports et adaptateurs), découpée en **features
étanches**, avec un **cœur écrit en style fonctionnel** : fonctions pures, valeurs immuables,
effets injectés et retournés.

- Les dépendances pointent vers l'intérieur, et cette flèche est **vérifiée mécaniquement**
  (`.arch-go.yml`, `depguard`).
- Une feature n'importe jamais une autre feature ; elles communiquent par événement
  ([ADR 006](006-outbox-transactionnel.md)).
- `domain/`, `ports/` et `application/` n'importent ni transport, ni persistance, ni logger.

## Conséquences

### Ce que ça achète

- Le métier se teste **sans conteneur, en microsecondes**. C'est le seul bénéfice qui compte, et il
  conditionne tous les autres : ce qui est facile à tester finit testé.
- Une feature se supprime, se réécrit ou s'extrait en un service sans toucher aux autres.
- Le remplacement d'une brique technique (base, transport, cache) est une opération locale.
- Une machine sans Docker suffit pour développer et vérifier le cœur.

### Ce que ça coûte

- **Plus de fichiers et plus d'indirection** qu'un `handler → service → repository`. Sur une
  feature de type CRUD, c'est un surcoût net et assumé : le socle optimise le cas où la logique
  existe, pas le cas où elle est absente.
- Deux features qui partagent une notion la **dupliquent**. C'est délibéré ([`rules/solid-et-dry.md`](../../rules/solid-et-dry.md)).
- Il faut écrire les ports avant l'implémentation, ce qui déplace l'effort de conception en amont.

### Ce que ça rend impossible

- Appeler la base depuis un cas d'usage sans passer par un port.
- Réutiliser une entité d'une feature dans une autre.
- « Juste ajouter un log » dans une règle métier.

## Alternatives écartées

| Alternative | Pourquoi écartée |
|---|---|
| Couches classiques (`handler`/`service`/`repository`) | Rien n'empêche le service d'importer le driver ; la règle n'est pas outillable |
| Clean Architecture avec interfaces | Produit des interfaces à N méthodes que personne n'implémente deux fois ; voir [ADR 003](003-ports-comme-types-fonction.md) |
| Monolithe sans découpe, découpe plus tard | « Plus tard » n'arrive pas : la découpe rétroactive coûte une refonte |
| Microservices dès le départ | Coût opérationnel sans bénéfice tant que l'équipe et le trafic ne l'exigent pas ; la découpe en features garde la porte ouverte |

## Garde

`.arch-go.yml` (matrice d'imports, `shouldNotContainInterfaces`, limites de taille de fonction),
`depguard` dans `.golangci.yml` (paquets interdits par couche), job CI `architecture`.
