# Parité fonctionnelle avec Spring Boot, Laravel et Symfony

> Ce document cadre le périmètre du framework. Il distingue rigoureusement
> **✅ prouvé** / **⚠️ écrit non prouvé** / **🔴 absent** / **🔜 version future** /
> **⛔ hors périmètre assumé**.
>
> Il n'a pas le droit de mentir : c'est lui qu'on lira pour décider si le socle
> est utilisable. Mis à jour le 2026-07-25.

## Ce que les frameworks matures donnent d'emblée

| Capacité | Spring Boot | Laravel | Symfony | Notre socle |
|---|---|---|---|---|
| **Génération de projet** | Initializr / CLI | `artisan make:*` | MakerBundle | 🔴 décidé, non écrit |
| **Commandes console** | `CommandLineRunner` | `artisan` | Console | 🔴 `cmd/cli` prévu, absent |
| **Injection de dépendances** | conteneur | conteneur | conteneur | ⛔ **remplacée** par composition manuelle ([ADR 004](../adr/004-composition-manuelle-sans-conteneur-di.md)) |
| **Configuration multi-env** | profiles | `config/` + `.env` | `config/packages/` | ⚠️ `config/` groupé + `env/` — non prouvé |
| **Routage** | `@RequestMapping` | `routes/` | attributs | ⚠️ chi + huma — non prouvé |
| **Validation** | Bean Validation | Validator | Validator | ⚠️ huma déclaratif + constructeurs intelligents |
| **Accès aux données** | JPA / Hibernate | Eloquent | Doctrine | ⚠️ pgx + sqlc ([ADR 009](../adr/009-strategie-d-acces-aux-donnees.md)) ; ORM confiné |
| **Migrations** | Flyway / Liquibase | `artisan migrate` | Doctrine Migrations | 🔴 goose choisi, **fichiers absents** |
| **Seeders / fabriques** | `data.sql` | seeders + factories | DoctrineFixtures | 🔴 décidé (via les cas d'usage), non écrit |
| **Files et travaux** | `@Async`, Batch | queues / jobs | Messenger | ⚠️ outbox + worker + relais Kafka/AMQP |
| **Tâches planifiées** | `@Scheduled` | `schedule()` | cron | ⚠️ verrou consultatif Postgres |
| **Événements** | `ApplicationEvent` | events / listeners | EventDispatcher | ⚠️ outbox + bus inproc/broker |
| **Cache** | `@Cacheable` | facade Cache | composant Cache | ⚠️ décorateur + Redis |
| **Courriel** | `JavaMailSender` | facade Mail | Mailer | 🔴 port défini, **adaptateur absent** |
| **Internationalisation** | `MessageSource` | `lang/` | Translation | 🔴 configuration écrite, **paquet absent** |
| **Authentification / autorisation** | Spring Security | Sanctum / Gates | SecurityBundle | 🔴 **RIEN — le plus gros manque** |
| **Drapeaux de fonctionnalité** | — | Pennant | — | ⚠️ `dynconf` |
| **Observabilité** | Actuator / Micrometer | Telescope | — | ⚠️ OpenTelemetry + sinks configurables |
| **Sondes de santé** | Actuator | — | — | ⚠️ `/healthz`, `/readyz` |
| **Documentation d'API** | springdoc | Scribe | NelmioApiDoc | ⚠️ huma → OpenAPI généré |
| **Pagination** | `Pageable` | `paginate()` | KnpPaginator | ⚠️ par curseur, pas par `OFFSET` |
| **Stockage de fichiers** | `Resource` | facade Storage | Flysystem | ⚠️ port + adaptateur disque |
| **Limitation de débit** | — | RateLimiter | RateLimiter | ⚠️ en mémoire, **par instance** |
| **Sérialisation / vues** | Jackson | API Resources | Serializer | ⚠️ un présentateur par surface |
| **Tests et fabriques** | Spring Test | PHPUnit + factories | PHPUnit | 🔴 **zéro test** |
| **Multi-tenant** | — | — | — | ⚠️ RLS Postgres + portée transactionnelle |
| **Idempotence des écritures** | — | — | — | ⚠️ **au-delà de la parité** |
| **Isolation par schéma** | — | — | — | ⚠️ **au-delà de la parité** |
| **Administration / CRUD auto** | — | Nova | EasyAdmin | ⛔ hors périmètre |
| **Microservices** | Spring Cloud | — | — | 🔜 version future |

## Ce qu'il manque pour atteindre la parité

Par ordre de coût d'absence, pas par ordre de difficulté :

1. **Authentification et autorisation.** C'est le seul manque qui rend le socle
   inutilisable en production. Aucun des trois frameworks cités ne se concevrait
   sans. Doit devenir le premier module réel, avec sa matrice d'accès et ses
   tests de refus.
2. **Tests.** Zéro aujourd'hui. Un framework sans tests ne propage pas de la
   productivité, il propage ses défauts.
3. **Migrations et seeders.** Six tables sont référencées par du code écrit et
   n'existent dans aucune migration.
4. **Commandes console.** L'équivalent d'`artisan` est ce qui rend un framework
   utilisable au quotidien : migrations, seeds, inspection, tâches ponctuelles.
5. **Internationalisation.** La configuration existe, le paquet non. Le domaine
   ne rend qu'un `ErrorCode` — la traduction appartient à chaque surface.
6. **Gabarits de courriel.** Port défini, adaptateur absent.
7. **Génération de projet et de feature.** Ce qui transforme le socle en
   framework.

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
