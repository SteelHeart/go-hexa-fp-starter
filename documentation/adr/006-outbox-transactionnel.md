# ADR 006 — Outbox transactionnel comme seule sortie vers le monde

- **Statut** : Accepté
- **Date** : 2026-07-25

## Contexte

Les features sont étanches ([ADR 001](001-hexagonal-modulaire-et-fonctionnel.md)) : elles ne
s'importent pas et ne partagent pas de table. Il leur faut donc un moyen de se coordonner.

Publier directement depuis un cas d'usage crée une **écriture double non atomique** :

- publication réussie, transaction annulée → le monde croit à un fait qui n'a pas eu lieu ;
- transaction validée, publication échouée → le fait a eu lieu et personne ne le sait.

Le second cas est le plus dangereux : il est silencieux.

## Décision

Le seul chemin de sortie est une **table `outbox_messages`, écrite dans la même transaction** que
le changement d'état.

```
transaction : INSERT users …  +  INSERT outbox_messages …     ← atomique
   worker    : SELECT … FOR UPDATE SKIP LOCKED → publie → marque traité
```

- Un cas d'usage écrit via le port `PublishEvent`. Il ne connaît aucun broker, aucune file.
- Le dépilage est **au moins une fois** : tout consommateur est **idempotent**, et cette
  idempotence est testée.
- `FOR UPDATE SKIP LOCKED` permet à plusieurs workers de tourner sans coordination.
- Un message en échec est réessayé avec recul exponentiel jusqu'à `WORKER_MAX_ATTEMPTS`, puis
  marqué `failed` — **jamais supprimé**.
- Le contexte de trace est transporté dans l'enveloppe : la consommation asynchrone se rattache à
  la requête qui l'a produite.

## Conséquences

### Ce que ça achète

- **Aucune perte, aucun fantôme.** L'atomicité est celle de la base, pas celle d'un protocole
  distribué.
- Un broker indisponible n'empêche pas d'accepter des écritures : les messages s'accumulent.
- La table est **inspectable** : en incident, on lit du SQL, pas des offsets.
- La découpe en features tient sans dépendance directe.

### Ce que ça coûte

- **Cohérence à terme** : la feature cible est en retard sur la source, de quelques centaines de
  millisecondes à plusieurs secondes. Ce délai doit être acceptable pour le métier — sinon les
  deux features n'en font qu'une, et c'est une décision légitime.
- Un **worker à opérer**, à superviser, et dont la mort est silencieuse sans métrique.
  `outbox_pending_messages` est pour cette raison la métrique la plus importante du système.
- Charge d'écriture supplémentaire, et une table à purger.
- L'idempotence des consommateurs est une contrainte permanente, pas une option.

### Ce que ça rend impossible

- Publier un événement depuis un cas d'usage.
- Faire dépendre une feature de la réussite immédiate d'une autre.

## Alternatives écartées

| Alternative | Pourquoi écartée |
|---|---|
| Publication directe depuis le cas d'usage | Écriture double non atomique : perte ou fantôme garantis |
| Appel synchrone entre features | Recrée le couplage que la découpe visait à supprimer |
| *Change data capture* (Debezium…) | Aucune dépendance applicative, mais un composant lourd à opérer ; réévaluable au-delà d'un certain volume |
| File de messages transactionnelle native | Impose le choix du broker au socle ; l'outbox reste agnostique |

## Garde

Revue humaine (aucun linter ne détecte une publication directe). Test d'idempotence obligatoire par
consommateur ([`rules/definition-of-done.md`](../../rules/definition-of-done.md) § 5).
