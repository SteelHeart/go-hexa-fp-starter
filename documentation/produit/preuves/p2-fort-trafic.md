# P2 — Le développeur backend à fort trafic

> **Verdict : 🔴 quatre critères sur cinq rouges.** Un est passé au vert, et un autre s'est révélé
> **pire que ce que la grille annonçait**.

Mesuré le **2026-07-29**, sur `main`, dans la toolbox.

## Ce que P2 voulait faire

La même liste de tâches que P1, mais avec ce qu'elle exige : **import en masse** au-delà de 1 MiB,
**export en flux**, et un **traitement long hors requête**.

## Le tableau de critères de P2, remesuré

| Critère | Cible | Mesuré le 2026-07-29 | Verdict |
|---|---|---|---|
| Réponse en flux possible | oui | **non** — aucun `http.Flusher`, aucun `text/event-stream` dans tout le dépôt | 🔴 |
| Taille d'ingestion | configurable | **non** — et pas pour la raison écrite, voir ci-dessous | 🔴 |
| File de travaux longs | oui | **aucune** — les huit modules noyau ne comportent aucune file généraliste ; `cmd/worker` ne dépile que l'outbox | 🔴 |
| Benchmarks de non-régression | ≥ 1 | **`tests/perf/` existe** et tourne (#91) — mais **0 `func Benchmark`** en Go | ⚠️ |
| Profilage mémoire sous charge | possible | **aucun `pprof`, aucun `GOMEMLIMIT`, aucun `GOMAXPROCS`** | 🔴 |

## 🔴 La découverte : `max_body_bytes` ne fait rien

La grille dit : *« Taille d'ingestion : **1 MiB en dur** (`max_body_bytes`) »*.

**C'est faux sur la cause, et la cause change le remède.**

`config/local.yaml` réglé à 50 MiB, puis un corps de 5 MiB :

```
statut: 413
{"title":"Request Entity Too Large","status":413,
 "detail":"request body is too large limit=1048576 bytes"}
```

`1 048 576` = 1 MiB. **Ce n'est pas la valeur configurée.** Le routeur pose **deux** bornes :

```go
middleware.MaxBody(cfg.HTTP.MaxBodyBytes)                      // configurable — et jamais atteinte
humaCfg := huma.DefaultConfig(cfg.App.Name, cfg.App.Version)   // porte SA borne, non reliée
```

Sur une route huma — c'est-à-dire **toutes les routes métier** — c'est la seconde qui parle en
premier.

**Une clé de configuration qui existe, est documentée, est validée au démarrage, et n'agit pas.**
C'est pire qu'une valeur en dur : une valeur en dur se voit. Celle-ci fait croire que la question est
réglée.

L'exploitant qui relève la limite verra sa configuration acceptée, le service démarrer, et les
requêtes échouer avec une valeur qu'il n'a écrite nulle part.

→ **Issue [#141](https://github.com/SteelHeart/go-hexa-fp-starter/issues/141).** Le remède est d'une
ligne — plus le test sans lequel il n'est pas vérifiable.

## Le critère qui a bougé

**Benchmarks de non-régression** passe de 🔴 à ⚠️. `tests/perf/` existe désormais et a tourné : **3 483
inscriptions, 100 % de 201, p95 à 288 ms, 58 req/s** — le coût d'Argon2id (#91).

Ce n'est pas encore le vert : la cible dit « ≥ 1 benchmark », et il y a **zéro `func Benchmark`** en
Go. Un scénario k6 mesure le service **de l'extérieur**, un benchmark Go mesure une fonction. Les
deux sont utiles, et P2 demande le second.

⚠️ **Et le scénario k6 a lui-même révélé un piège pour P2** : à sa première exécution il rendait
**26 027 req/s** — le débit auquel le **limiteur refuse**, pas celui auquel le socle inscrit. Un
chiffre de charge sans son contexte est un chiffre faux, et P2 est précisément la persona qui les
lira.

## Ce que P2 a pu faire

Créer son projet et ses modules, comme P1 — en 6 secondes. **Puis s'arrêter.**

- Pas d'import en masse : 413 au-delà de 1 MiB, sans recours par la configuration ;
- pas d'export en flux : rien dans le dépôt ne sait produire une réponse incrémentale ;
- pas de traitement long : `cmd/worker` ne dépile que l'outbox, et l'outbox n'est pas une file de
  travaux — c'est un journal de publication.

## Ce qu'il aurait fallu contourner

Tout, et par du code hors du socle. C'est le point : **P2 devrait écrire l'infrastructure que le
framework promet de lui éviter d'écrire.**

## Le constat qui compte, et il n'est pas technique

> **Cinq relevés, et P2 n'a reçu aucun gain de conception.**

Les seuls mouvements de cette persona depuis le premier relevé sont : un critère passé de 🔴 à ⚠️ par
un lot qui la visait indirectement (#91), et un rouge dont on comprend **enfin la vraie cause**
(#141).

### Pourquoi, et ce n'est pas ce que je croyais en écrivant la ligne précédente

Ce document a d'abord conclu : *« une suite de lots individuellement justifiés a composé un ordre de
priorité que personne n'a choisi. »* La formulation est fausse, et flatteuse.

Vérifié le **2026-07-29**, sur les 23 issues ouvertes et sur l'ensemble des issues fermées :

```
stream · SSE · Flusher · temps reel     → 0
queue · file de travaux                  → 0
benchmark                                → 0
pprof · GOMEMLIMIT · memoire             → 0
configuration fermee · extension         → 0
```

**Aucun des cinq critères de P2 n'avait d'issue. Pas une seule.** Une seule issue de tout le dépôt
citait P2 avant ce jour, et elle est fermée (#91).

Il n'y avait donc **pas d'ordre de priorité**, choisi ou non : il n'y avait **rien à prioriser**. Un
lot dépriorisé figure dans une liste, se voit et se discute ; un lot qui n'existe pas ne figure
nulle part. C'est pour cela que cinq relevés ont pu constater le même rouge sans que rien ne bouge —
et c'est la famille de défaut que ce dépôt traque partout ailleurs : **une absence qui ne produit
aucun symptôme**.

Les cinq lots ont été ouverts le 2026-07-29 : **#143** (flux), **#144** (file de travaux), **#145**
(benchmarks), **#146** (mémoire et profilage), **#147** (point d'extension de la configuration).
#144 et #147 portent `needs-decision` : ils engagent l'architecture et attendent un arbitrage.

**Ce que P2 tue** : *« la configuration fermée et les délais HTTP en dur. Les deux sont
rédhibitoires pour elle, pas gênants. »* Les deux tiennent toujours.
