# Matrice des pilotes

> Surface d'extension du framework. Un module déclare des **ports** ; un **pilote** en est une
> implémentation interchangeable, choisie par configuration.
>
> **Ce document est un catalogue d'INTENTIONS, pas un inventaire.** Il définit le périmètre
> d'extension, pas l'existant.

## Ce qui est réellement construit

Une centaine de pilotes sont décrits plus bas. **Onze existent.** La table qui fait autorité sur ce
qu'on peut activer est `knownDrivers`, dans
[`internal/config/modules.go`](../../internal/config/modules.go) : elle ne liste que les pilotes
écrits, testés, et qui documentent leurs NON-garanties.

| Module noyau | Pilotes construits |
|---|---|
| `outbox` | `memory` · `postgres` |
| `idempotency` | `memory` · `postgres` · `redis` |
| `dynconf` | `file` · `postgres` |
| `audit` | `log` · `postgres` |
| `storage` | `disk` |
| `scheduler` | `cron-inproc` · `advisory-lock` |

Tout le reste **refuse le démarrage**. C'est délibéré : accepter `driver: s3` dans la configuration
puis échouer plus loin avec « pilote inconnu » ferait se contredire deux sources de vérité, et le
message accuserait l'utilisateur d'une faute qui serait la nôtre.

Un pilote migre de ce catalogue vers `knownDrivers` le jour où il est écrit, testé, et où il déclare
ce qu'il ne garantit pas — jamais avant.

> Les pilotes `postgres` et `redis` ci-dessus sont **écrits mais jamais exécutés** : aucune migration
> n'existe (issue #2) et aucun service ne tourne sur la machine de référence (friction F001). Ils
> compilent, ils ont été relus, rien ne les a éprouvés.

## Six règles qui gouvernent tout pilote

**1. Chaque module a un pilote par défaut sans dépendance externe.**
`memory`, `file`, `log`, `disk`, `builtin`. C'est ce qui permet à `hexa new` de produire une
application qui démarre sans base, sans Redis, sans Docker ([ADR 012](../adr/012-anatomie-d-un-module-et-pilotes.md)).

**2. Les pilotes lourds sont des modules Go séparés.**
Sinon chaque projet télécharge et compile tous les SDK du catalogue. Nous en portons déjà le défaut :
`kafka-go` et `amqp091-go` sont dans le `go.mod` principal alors que le pilote par défaut est
`inproc`. À corriger lors du passage en monorepo multi-modules.

```
core/                             aucun SDK de fournisseur, jamais
drivers/storage-s3/               module Go distinct
drivers/payment-stripe/
```

**3. Un pilote ne change JAMAIS une signature de port.** S'il en a besoin, ce n'est pas un pilote :
c'est un nouveau port. C'est la règle qui empêche le catalogue de faire fuir les fournisseurs dans
le cœur.

**4. Tout pilote passe la même suite de conformité.** Y compris sur le **contrat d'erreur** du port.
Un pilote `memory` qui ne reproduirait pas la violation d'unicité du pilote `postgres` mentirait sur
le comportement de production.

**5. Un pilote documente ses NON-garanties.** `outbox/memory` ne survit pas à un redémarrage.
`storage/disk` n'est pas partagé entre répliques. `ratelimit/memory` se multiplie par le nombre
d'instances. Une garantie supposée à tort est plus dangereuse qu'une absence de garantie.

**6. Un pilote inconnu refuse le démarrage.** Deny par défaut, jusque dans la lecture de la
configuration.

---

## `auth` — identité et autorisation

| Axe | Pilotes |
|---|---|
| **Stratégie** | `password` (Argon2id) · `oauth2` · `oidc` · `saml2` · `ldap` · `apikey` · `mtls` · `magiclink` · `webauthn` (passkeys) |
| **Fournisseur OIDC** | `google` · `microsoft-entra` · `apple` · `github` · `gitlab` · `okta` · `auth0` · `keycloak` · `zitadel` · `authentik` · `cognito` · `firebase` · `generic` |
| **SSO SAML** | `adfs` · `azure-ad` · `okta` · `onelogin` · `generic` |
| **Jeton** | `opaque` (défaut, révocable) · `jwt-hs256` · `jwt-rs256` · `jwt-es256` · `paseto` |
| **Magasin de session** | `memory` · `cookie` (signé et chiffré) · `redis` · `postgres` |
| **Autorisation** | `rbac` (défaut) · `permissions` (table) · `abac` · `casbin` · `opa` (Rego) · `openfga` / `spicedb` (ReBAC) |

Le jeton **opaque** est le défaut, et pas le JWT : un JWT n'est pas révocable avant expiration. Un
JWT est un choix de performance, jamais un défaut de sécurité acceptable.

## `notification` — canaux × fournisseurs

| Canal | Pilotes |
|---|---|
| **mail** | `log` (défaut) · `smtp` · `mailjet` · `sendgrid` · `ses` · `postmark` · `mailgun` · `resend` · `brevo` · `mailpit` (dev) |
| **sms** | `log` · `twilio` · `vonage` · `messagebird` · `infobip` · `africastalking` · `orange-sms` |
| **push** | `log` · `fcm` · `apns` · `expo` · `onesignal` · `webpush` (VAPID) |
| **conversation** | `slack` · `teams` · `telegram` · `whatsapp-business` · `discord` |
| **dans l'application** | `memory` · `postgres` · `redis` |
| **totp** | `builtin` (RFC 6238) — aucun fournisseur externe, et c'est voulu |

## `payment` — encaissement

| Famille | Pilotes |
|---|---|
| **Carte / portefeuille** | `log` (défaut, refusé hors développement) · `stripe` · `adyen` · `paypal` · `mollie` · `braintree` · `checkout` · `square` |
| **Mobile money (UEMOA/Afrique)** | `orange-money` · `mtn-momo` · `moov` · `wave` · `kkiapay` · `cinetpay` · `paydunya` · `fedapay` |
| **Bancaire** | `sepa-dd` · `gocardless` · `virement` (rapprochement manuel) |
| **Registre** | `postgres` (obligatoire, ajout seul) |

## `storage` / `media` — objets et fichiers

| Axe | Pilotes |
|---|---|
| **Objet** | `disk` (défaut) · `memory` (tests) · `minio` · `s3` · `gcs` · `azure-blob` · `r2` (Cloudflare) · `b2` (Backblaze) · `wasabi` · `scaleway` · `ovh` · `sftp` · `webdav` |
| **Transformation** | `none` (défaut) · `builtin` (Go pur) · `imgproxy` · `thumbor` · `cloudinary` |
| **Analyse antivirale** | `none` (défaut) · `clamav` · `virustotal` |
| **URL signée** | `builtin` (HMAC) · `cloudfront` · `cloudflare` |

Les pilotes `minio`, `s3`, `r2`, `b2`, `wasabi`, `scaleway` et `ovh` partagent l'API S3 : **un seul
pilote `s3` paramétré par point d'accès** suffit, plutôt que sept. C'est le genre de mutualisation
qu'il faut faire, par opposition au sur-DRY entre modules.

## `search` — indexation et recherche

`postgres-fts` (défaut) · `sqlite-fts` · `bleve` (embarqué, Go pur) · `meilisearch` · `typesense` ·
`opensearch` · `elasticsearch` · `algolia`

**Sémantique / vectoriel** : `pgvector` · `qdrant` · `weaviate` — port distinct (`SemanticSearch`),
pas un pilote de `Query` : les garanties ne sont pas les mêmes.

## `messaging` — relais d'événements

`inproc` (défaut) · `noop` · `postgres` (LISTEN/NOTIFY) · `kafka` · `rabbitmq` · `nats` ·
`redis-streams` · `sqs` · `gcp-pubsub` · `azure-servicebus` · `mqtt`

## `cache`

`memory` (défaut) · `redis` · `valkey` · `memcached` · `dragonfly` · `postgres-unlogged`

## `secrets` — module manquant, à ajouter

Nous lisons les secrets depuis l'environnement, ce qui suffit aujourd'hui mais ne couvre ni la
rotation ni l'audit d'accès.

`env` (défaut) · `file` · `sops` · `vault` (HashiCorp) · `aws-secrets-manager` ·
`gcp-secret-manager` · `azure-key-vault` · `infisical`

## `dynconf` — drapeaux et réglages

`file` (défaut) · `postgres` · `redis` · `consul` · `etcd` · `unleash` · `flagsmith` · `launchdarkly`

## `idempotency`

`memory` (défaut) · `postgres` · `redis`

## `audit`

`log` (défaut) · `postgres` · `s3-worm` (verrouillage d'objet) · `kafka`

## `tenancy` — multi-locataire

| Pilote | Isolation | Coût |
|---|---|---|
| `rls` (défaut) | Colonne `tenant_id` + *Row Level Security* | Une base, une migration. Une erreur de politique fuit |
| `schema` | Un schéma Postgres par locataire | Isolation forte. Les migrations se multiplient par le nombre de locataires |
| `database` | Une base par locataire | Isolation maximale. Coût opérationnel élevé, agrégations inter-locataires impossibles |

## `workflow` — machine à états

`builtin` (déclaratif, défaut) · `temporal` · `cadence`

## `document` — rendu

`html` (défaut) · `gotenberg` · `weasyprint` · `chromedp` (headless) · `typst` · `latex`

## `export` / `import`

`csv` (défaut) · `xlsx` · `json` · `jsonl` · `parquet` · `pdf` (via `document`)

## `webhook` — sortant

`outbox` (défaut, livraison HTTP signée HMAC) · `svix` (géré)

## `scheduler`

`advisory-lock` (défaut) · `cron-inproc` · `temporal` · `external` (CronJob Kubernetes)

## `ratelimit`

`memory` (défaut, **par instance**) · `redis` · `postgres` · `gateway` (délégué au mandataire)

## `observability`

| Signal | Pilotes |
|---|---|
| **journaux** | `stdout` (défaut) · `file` — la collecte appartient à l'agent, pas à l'application |
| **traces** | `otlp-grpc` (défaut) · `otlp-http` · `none` |
| **métriques** | `prometheus` (défaut, tirage) · `otlp` (poussée) · `none` |
| **erreurs** | `none` (défaut) · `sentry` · `glitchtip` · `rollbar` · `bugsnag` |

ELK, Grafana, Datadog et Honeycomb **ne sont pas des pilotes** : ils consomment du JSON sur stdout
et de l'OTLP. Changer d'outil de supervision se fait dans l'infrastructure, jamais dans
l'application.

## `i18n`

`embedded` (défaut, `go:embed`) · `file` · `postgres` · synchronisation `crowdin` / `weblate`

## `consent` — RGPD / APDP

`postgres` (défaut). Pas de pilote externe : la cartographie des données personnelles est
structurelle et propre au projet. Ports : `RecordConsent` · `RevokeConsent` · `ExportSubjectData` ·
`EraseSubjectData`.

---

## Ce qui n'entrera pas au catalogue

| Écarté | Motif |
|---|---|
| ORM alternatif | [ADR 009](../adr/009-strategie-d-acces-aux-donnees.md) |
| CRUD administratif généré | Contourne les cas d'usage, donc les règles métier |
| CMS | Un produit, pas une brique de framework |
| Système de plugins chargés à l'exécution | Go compile statiquement ; les pilotes couvrent le besoin |
| Temps réel (WebSocket, SSE) | Ce n'est **pas un module** mais une **surface** : `surfaces/ws/` |
