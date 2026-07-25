# Observabilité

Un système qu'on ne peut pas interroger en production n'est pas fini. Ce qui suit est le minimum,
pas l'objectif.

## 1. Les trois signaux, et leur rôle

| Signal | Question à laquelle il répond | Outil |
|---|---|---|
| **Trace** | *Où* le temps est passé, dans quel ordre | OpenTelemetry → OTLP |
| **Métrique** | *Combien*, et depuis quand ça dérive | OpenTelemetry → Prometheus |
| **Log** | *Ce qui s'est exactement passé* sur une requête | `log/slog`, JSON |

Les trois sont reliés par le **`trace_id`**. Un log sans `trace_id` est un log qu'on ne pourra pas
recouper : le middleware l'injecte, et il ne doit jamais être omis.

## 2. Le cœur ne journalise pas

`domain/`, `ports/` et `application/` **n'importent pas `log/slog`** (`depguard` échoue la CI).

Le cœur **retourne** son résultat ; ce sont les **décorateurs** qui observent :

```go
register = application.Apply(register,
    application.WithTransaction(runInTx),
    application.WithTracing(tracer),   // ouvre un span, enregistre l'issue
    application.WithLogging(logger),   // journalise entrée/sortie/durée
)
```

Bénéfice concret : un test du cas d'usage n'a pas de logger, pas de tracer, pas de sortie parasite.
Et l'instrumentation s'ajoute ou se retire sans toucher au métier.

## 3. Traces

- Un span par cas d'usage, nommé `{module}.{usecase}` — pas le nom de la fonction Go, qui change
  au refactoring.
- Attributs : identifiants métier, jamais de valeur sensible.
- Une erreur métier n'est **pas** un span en erreur : `CodeEmailAlreadyExists` est un
  fonctionnement nominal. Seuls `CodeUnavailable` et `CodeInternal` marquent le span en erreur.
  Sans cette distinction, le taux d'erreur devient inexploitable.
- Le contexte de trace est propagé **jusque dans l'outbox** : un événement consommé par le worker
  doit se rattacher à la requête qui l'a produit, sinon la chaîne asynchrone est aveugle.

## 4. Métriques

Quatre familles suffisent au démarrage :

| Métrique | Type | Dimensions |
|---|---|---|
| `http_server_request_duration` | histogramme | route, méthode, statut |
| `usecase_duration` | histogramme | module, cas d'usage, issue (`ok` / code d'erreur) |
| `outbox_pending_messages` | jauge | — |
| `outbox_message_attempts` | compteur | type d'événement, issue |

**Aucune dimension à cardinalité non bornée** : jamais d'identifiant utilisateur, d'email ni de
chemin brut en étiquette. C'est la première cause d'explosion d'un stockage de métriques.

`outbox_pending_messages` est la métrique la plus importante du système : elle monte quand le
worker est mort, et c'est le seul symptôme visible d'une chaîne asynchrone en panne.

## 5. Logs

- **JSON structuré** en UAT et en production. Format lisible en développement local.
- Champs obligatoires : `time`, `level`, `msg`, `trace_id`, `span_id`, `service`, `env`.
- Un log par requête en sortie, plus les erreurs. Pas de log de progression dans une boucle chaude.
- **Jamais de donnée personnelle en clair** ([`securite.md`](securite.md) § 5).
- Niveaux : `Error` = action humaine requise. `Warn` = anomalie absorbée. `Info` = fait métier.
  `Debug` = désactivé hors développement. Un `Error` qui n'appelle aucune action est un `Warn`.

## 6. Sondes

| Route | Sémantique | Vérifie |
|---|---|---|
| `/healthz` | le processus est vivant | rien d'externe — sinon un incident base tue tous les conteneurs |
| `/readyz` | le processus peut servir | base et cache joignables, avec délai court |
| `/metrics` | exposition Prometheus | port séparé, **jamais** exposé publiquement |

## 7. Ce qui reste à faire côté opérations

Les alertes (`outbox_pending_messages` qui croît, taux de `CodeInternal`, latence p99) ne sont pas
dans ce dépôt : elles appartiennent au socle d'infrastructure. **Écrit ici, non déployé** — la
distinction est explicite conformément à [`README.md`](README.md) § règle d'or 2.
