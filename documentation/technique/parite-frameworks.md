# Parité fonctionnelle avec Spring Boot, Laravel et Symfony

> Ce document cadre le périmètre du framework. Il distingue rigoureusement
> **✅ prouvé** / **⚠️ écrit non prouvé** / **🔴 absent** / **🔜 version future** /
> **⛔ hors périmètre assumé**.
>
> Il n'a pas le droit de mentir : c'est lui qu'on lira pour décider si le socle
> est utilisable. **Vérifié ligne à ligne le 2026-07-27.**

## ⚠️ À lire avant le tableau — ce que « prouvé » veut dire ici

**Aucune intégration continue n'a jamais tourné sur ce dépôt** : 66 exécutions, 66
`startup_failure`, zéro succès depuis le premier commit (issue #47). Tout ce qui
porte ✅ ci-dessous a donc été vérifié **en local**, dans l'environnement
reproductible de [`deploy/toolbox/`](../../deploy/toolbox/README.md), commande
`task ci` — 10 des 12 jobs, codes de retour vérifiés.

C'est un niveau de preuve réel, et inférieur à celui d'une CI : il ne dit rien
d'une autre machine, et il est contournable. La distinction est maintenue
partout dans ce document.

## Ce que les frameworks matures donnent d'emblée

| Capacité | Spring Boot | Laravel | Symfony | Notre socle |
|---|---|---|---|---|
| **Génération de projet** | Initializr / CLI | `artisan make:*` | MakerBundle | 🔴 décidé, non écrit (#17) |
| **Commandes console** | `CommandLineRunner` | `artisan` | Console | 🔴 `cmd/cli` prévu (#8), absent — `cmd/` ne contient que `server` et `worker` |
| **Injection de dépendances** | conteneur | conteneur | conteneur | ⛔ **remplacée** par composition manuelle ([ADR 004](../adr/004-composition-manuelle-sans-conteneur-di.md)) |
| **Configuration multi-env** | profiles | `config/` + `.env` | `config/packages/` | ✅ `config/` groupé + `env/` — 36 tests, et le serveur démarre sans aucun service |
| **Routage** | `@RequestMapping` | `routes/` | attributs | ✅ chi + huma — exercé de bout en bout |
| **Validation** | Bean Validation | Validator | Validator | ✅ huma déclaratif + constructeurs intelligents — un mot de passe court rend **422** avec le message du **domaine** |
| **Accès aux données** | JPA / Hibernate | Eloquent | Doctrine | ⚠️ **pgx en accès direct**, SQL écrit à la main ([ADR 009](../adr/009-strategie-d-acces-aux-donnees.md)). *`sqlc` n'est utilisé nulle part — la mention précédente était fausse.* ORM interdit |
| **Migrations** | Flyway / Liquibase | `artisan migrate` | Doctrine Migrations | ✅ goose — **appliquées**, `Down` réellement exécuté puis rejoué, invariants d'isolation vérifiés |
| **Seeders / fabriques** | `data.sql` | seeders + factories | DoctrineFixtures | 🔴 décidé (via les cas d'usage), non écrit |
| **Files et travaux** | `@Async`, Batch | queues / jobs | Messenger | ⚠️ outbox + dépileur : pilote `memory` **prouvé** (25 tests) ; pilote `postgres` et relais Kafka/AMQP **jamais exécutés** (#37) |
| **Tâches planifiées** | `@Scheduled` | `schedule()` | cron | ⚠️ pilote `cron-inproc` **prouvé** (16 tests) ; `advisory-lock` jamais exécuté. Module livré **désactivé** |
| **Événements** | `ApplicationEvent` | events / listeners | EventDispatcher | ⚠️ bus `inproc` couvert (23 tests). **Aucun consommateur ne s'abonne** : `user.registered.v1` est publié vers personne |
| **Cache** | `@Cacheable` | facade Cache | composant Cache | 🔴 compile, **zéro test à aucun niveau** — `New` fait un `Ping` Redis (#37) |
| **Courriel** | `JavaMailSender` | facade Mail | Mailer | 🔴 **aucun adaptateur** — pas un seul usage de `net/smtp` dans le dépôt |
| **Internationalisation** | `MessageSource` | `lang/` | Translation | 🔴 configuration écrite, **paquet absent** (#12) |
| **Authentification / autorisation** | Spring Security | Sanctum / Gates | SecurityBundle | 🔴 **RIEN — le plus gros manque** (#11) |
| **Drapeaux de fonctionnalité** | — | Pennant | — | ✅ `dynconf`, pilote `file` — 14 tests, deny par défaut prouvé |
| **Observabilité** | Actuator / Micrometer | Telescope | — | 🔴 configuration écrite et **`telemetry.Setup` appelée nulle part** (#13). Le câblage n'existe pas |
| **Sondes de santé** | Actuator | — | — | ✅ `/healthz` et `/readyz` — 200 constaté, 14 tests |
| **Documentation d'API** | springdoc | Scribe | NelmioApiDoc | ⚠️ contrat **servi** sur `/openapi.{json,yaml}` ; `api/openapi.yaml` **absent** et non versionné |
| **Pagination** | `Pageable` | `paginate()` | KnpPaginator | ✅ par curseur, pas par `OFFSET` — 11 tests |
| **Stockage de fichiers** | `Resource` | facade Storage | Flysystem | ✅ pilote `disk` — 13 tests, traversée de répertoire refusée en lecture **et** en écriture |
| **Limitation de débit** | — | RateLimiter | RateLimiter | ⚠️ en mémoire, **par instance** — 12 tests. Le module `ratelimit` partagé n'existe pas |
| **Sérialisation / vues** | Jackson | API Resources | Serializer | ⚠️ un présentateur par surface — mais **une seule surface existe** (`http`), donc la règle n'est pas démontrée |
| **Tests et fabriques** | Spring Test | PHPUnit + factories | PHPUnit | ⚠️ **285 tests** de premier niveau, 268 fichiers, cliquets tenus (74 / 95 / 60 %). **Fabriques absentes** |
| **Multi-tenant** | — | — | — | 🔴 **aucune politique RLS écrite, module `tenancy` absent** (#23). *La ligne précédente annonçait « RLS Postgres + portée transactionnelle » — c'était une intention, pas un état* |
| **Idempotence des écritures** | — | — | — | ✅ pilote `memory` — 25 tests, exclusivité sous concurrence prouvée. **Au-delà de la parité** |
| **Isolation par schéma** | — | — | — | ✅ ADR 011 vérifiée par `deploy/postgres/verify.sql` : `NOINHERIT`, refus de DDL et journal en ajout seul **constatés**. **Au-delà de la parité** |
| **Administration / CRUD auto** | — | Nova | EasyAdmin | ⛔ hors périmètre |
| **Microservices** | Spring Cloud | — | — | 🔜 version future |

## Ce qui manque pour atteindre la parité

Par ordre de **coût d'absence**, pas par ordre de difficulté :

1. **Authentification et autorisation** (#11). Seul manque qui rend le socle
   inutilisable en production. Aucun des trois frameworks cités ne se concevrait
   sans. Doit devenir le premier module réel, avec sa matrice d'accès et ses
   tests de refus.
2. **Une intégration continue qui démarre** (#47). Les 285 tests, les cliquets et
   les gardes existent — et rien ne les exécute automatiquement. Un framework
   dont la barrière n'a jamais tourné ne propage pas de la confiance.
3. **Commandes console** (#8). L'équivalent d'`artisan` est ce qui rend un
   framework utilisable au quotidien. C'est aussi la **deuxième surface**, sans
   laquelle la promesse « N frontends » n'a qu'une seule instance.
4. **Gabarits de courriel et `notification`**. Port défini, adaptateur absent :
   rien ne part.
5. **Internationalisation** (#12). Configuration écrite, paquet absent.
6. **Seeders et fabriques de test.**
7. **Génération de projet et de feature** (#17). Ce qui transforme le socle en
   framework — mais après tout le reste, parce qu'un générateur qui engendre un
   projet sans authentification n'aide personne.

## Ce qu'on ne fera pas, et pourquoi

| Écarté | Motif |
|---|---|
| **Conteneur d'injection** | La composition manuelle est vérifiée par le compilateur ; un conteneur résout des types, or nos ports sont des fonctions de même signature ([ADR 004](../adr/004-composition-manuelle-sans-conteneur-di.md)) |
| **Active Record** | Fait fuir la persistance dans le domaine ; c'est le défaut que toute l'architecture vise à empêcher |
| **CRUD administratif généré** | Produit des écrans qui contournent les cas d'usage, donc les règles métier |
| **Système de plugins** | Personne n'en a jamais eu besoin au bon moment ; les décorateurs et les adaptateurs couvrent l'extensibilité |
| **Façades statiques** | Rendent les dépendances invisibles et les tests dépendants de l'ordre |

## Microservices — version future

Le socle n'est **pas** un framework de microservices aujourd'hui, et ne prétend
pas l'être. Mais la porte est délibérément laissée ouverte, et trois décisions
déjà prises en constituent les fondations :

| Fondation déjà posée | Ce qu'elle permettra |
|---|---|
| **Un schéma Postgres par module**, rôle SQL dédié | Extraire un module, c'est déplacer un schéma — pas démêler des jointures |
| **`modulebus`** : `inproc` → `http` par configuration | Un module devient un service sans changement de code |
| **Langage publié** (`internal/contracts/`) | Le contrat inter-module existe déjà, indépendant des types internes |
| **Outbox + relais interchangeable** | La communication asynchrone ne change pas quand le broker apparaît |

Ce qui manquera alors, et qui n'est pas dans le périmètre actuel : découverte de
services, disjoncteur, traçage distribué inter-processus (partiellement couvert
par la propagation OTLP déjà en place), passerelle d'API, configuration
distribuée, et surtout la **maturité opérationnelle** que ces briques exigent.

> Le principe qui gouverne cette section : on ne construit pas les microservices
> aujourd'hui, on s'interdit seulement de les rendre impossibles. La différence
> de coût entre les deux est considérable, et c'est ce qui rend la position
> tenable.

---

## Journal des corrections de ce document

Un document d'arbitrage qui a menti une fois doit dire **quand** et **sur quoi**,
sinon le lecteur ne sait pas quelle version il tient.

| Date | Correction |
|---|---|
| 2026-07-27 | « Migrations 🔴 fichiers absents » → écrites, appliquées et vérifiées |
| 2026-07-27 | « Tests 🔴 **zéro test** » → **285 tests** de premier niveau, cliquets tenus |
| 2026-07-27 | « Multi-tenant ⚠️ RLS Postgres » → **🔴** : aucune politique RLS n'est écrite, le module `tenancy` n'existe pas. C'était une intention présentée comme un état |
| 2026-07-27 | « Accès aux données : pgx + **sqlc** » → `sqlc` n'est utilisé nulle part |
| 2026-07-27 | « Observabilité ⚠️ » → **🔴** : `telemetry.Setup` n'est appelée nulle part |
| 2026-07-27 | « Courriel : port défini » → aucun adaptateur, aucun usage de `net/smtp` |
| 2026-07-27 | Ajout de l'avertissement : **aucune CI n'a jamais tourné** — ce que « prouvé » signifie ici |
