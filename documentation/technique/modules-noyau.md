# Catalogue des modules noyau

> Anatomie et règles communes : [ADR 012](../adr/012-anatomie-d-un-module-et-pilotes.md).
>
> Ce document est une **cible de conception**. La colonne *État* dit, module par module, ce qui
> existe réellement — et le relevé qui fait foi reste [`documentation/AMORCAGE.md`](../../../AMORCAGE.md).

## Vue d'ensemble

Six modules sont convertis à l'anatomie de l'ADR 012 : `domain/` pur, `ports/` en types fonction,
un pilote par magasin, `module.go` seul à connaître les pilotes. Ils sont activables par
`config/modules.yaml` et couverts par des tests. Les autres n'existent pas — et un module absent du
**catalogue** assemblé par le composition root **refuse d'être activé** (ADR 014), précisément pour
qu'on ne puisse pas croire le contraire.

| Module | Rôle | Pilote par défaut | Autres pilotes | État |
|---|---|---|---|---|
| `outbox` | Publication garantie | `memory` | `postgres` | ✅ converti · `memory` prouvé |
| `idempotency` | Écritures rejouables | `memory` | `postgres` · `redis` | ✅ converti · `memory` prouvé |
| `dynconf` | Drapeaux et réglages à chaud | `file` | `postgres` | ✅ converti · `file` prouvé |
| `audit` | Journal en ajout seul | `log` | `postgres` | ✅ converti · `log` prouvé |
| `storage` | Objets et fichiers | `disk` | — | ✅ converti · `disk` prouvé |
| `scheduler` | Tâches périodiques | `cron-inproc` | `advisory-lock` | ✅ converti · `cron-inproc` prouvé |
| `auth` | Identité et autorisation | — | `oauth2` · `oidc` · `saml` · `password` · `apikey` | 🔴 rien — arbitrages en attente (#9) |
| `notification` | Messages sortants | `log` | `smtp` · `mailjet` · `ses` · `twilio` · `fcm` · `totp` | 🔴 rien |
| `payment` | Encaissement et remboursement | `log` | `stripe` · `mobile_money` · autres | 🔴 rien |
| `i18n` | Traduction par surface | `embedded` | `file` · `postgres` | 🔴 configuration seule |
| `ratelimit` | Limitation de débit | `memory` | `redis` · `postgres` · `gateway` | 🔴 rien |
| `tenancy` | Multi-locataire | `rls` | `schema` · `database` | 🔴 rien (#23) |
| `secrets` | Rotation et audit d'accès | `env` | `vault` · `sops` · gestionnaires infonuagiques | 🔴 rien (#26) |
| `workflow` | Machine à états | `builtin` | `temporal` | 🔴 rien (#25) |
| `search` | Recherche | `postgres-fts` | `bleve` · `meilisearch` · `opensearch` | 🔴 rien (#29) |
| `document` | Rendu PDF | `html` | `gotenberg` · `weasyprint` · `chromedp` | 🔴 rien (#32) |

**« Prouvé » ne concerne que le pilote nommé.** Les pilotes `postgres` et `redis` des modules
convertis sont écrits et relus, **jamais exécutés** : aucune migration n'existe (#2) et aucun service
ne tourne sur la machine de référence (F001).

Le pilote par défaut du `scheduler` était `advisory-lock` : il exigeait donc une base pour
simplement répéter une tâche, y compris dans un binaire mono-instance qui n'a personne avec qui
s'accorder. C'est `cron-inproc` depuis la conversion — l'élection est devenue un pilote.

---

## `auth` — identité et autorisation

Le mode d'authentification **dépend de la surface servie**, jamais du cœur. Chaque stratégie est un
pilote ; le port reste unique.

```yaml
auth:
  enabled: true
  strategies:
    oauth2:   { enabled: true }            # défaut
    oidc:
      providers:
        google:    { client_id: ${GOOGLE_CLIENT_ID}, client_secret: ${GOOGLE_CLIENT_SECRET} }
        microsoft: { tenant: ${MS_TENANT}, client_id: ${MS_CLIENT_ID} }
    saml_sso: { enabled: false }
    password: { enabled: false }           # Argon2id — refusé par défaut
    apikey:   { enabled: true }            # surface CLI et intégrations
  store: { driver: postgres }              # où vit l'identité locale
  surfaces:
    web:    { session_cookie: true,  bearer: false }
    mobile: { session_cookie: false, bearer: true  }
    cli:    { apikey: true }
  authorization:
    model: rbac                            # rbac | abac | permissions
```

**Ports** : `Authenticate` · `Authorize` · `IssueSession` · `RevokeSession` · `LinkIdentity`.

Trois principes non négociables :

- **Le jeton authentifie, il n'autorise pas.** `Authorize` interroge l'état persisté, à chaque appel.
- **Deny par défaut** : toute stratégie non explicitement activée refuse ; tout modèle
  d'autorisation inconnu refuse le démarrage.
- L'identité entre dans un module métier comme **paramètre de commande**, jamais lue d'un contexte
  implicite.

`totp` **n'appartient pas à `auth`** mais à `notification` : `auth` le consomme par son langage
publié. Sinon on recrée le couplage que toute l'architecture combat.

---

## `notification` — canaux × fournisseurs

```yaml
notification:
  channels:
    mail: { provider: smtp }     # log | smtp | mailjet | ses | sendgrid
    sms:  { provider: none }     # none | twilio | vonage
    push: { provider: none }     # none | fcm | apns
    totp: { enabled: true }      # RFC 6238 — aucun fournisseur externe
  providers:
    smtp:    { addr: ${SMTP_ADDR}, from: ${SMTP_FROM} }
    mailjet: { api_key: ${MAILJET_KEY}, secret: ${MAILJET_SECRET} }
```

**Ports** : `Send` · `SendTemplate` · `GenerateTOTP` · `VerifyTOTP`.

- Un envoi passe **toujours** par l'outbox : sinon une notification part alors que la transaction
  métier a échoué.
- Un consommateur de notification est **idempotent** : le dépilage est « au moins une fois », donc
  un courriel sera rejoué au moins une fois dans la vie du système.
- Les gabarits sont embarqués et **jamais** interpolés avec du contenu non échappé.
- Aucune donnée personnelle dans les journaux : on trace l'identifiant du destinataire, pas son
  adresse.

---

## `payment` — le module où l'erreur coûte de l'argent réel

C'est le seul module où un doublon n'est pas un défaut d'affichage mais un débit en trop. Ses règles
sont donc plus strictes que partout ailleurs.

```yaml
payment:
  enabled: false
  provider: log                  # log | stripe | mobile_money | ...
  currency: XOF
  providers:
    stripe:
      secret_key:      ${STRIPE_SECRET_KEY}
      webhook_secret:  ${STRIPE_WEBHOOK_SECRET}
    mobile_money:
      operator: ${MOMO_OPERATOR}
      api_key:  ${MOMO_API_KEY}
  ledger: { driver: postgres }    # registre en ajout seul, obligatoire
```

**Ports** : `Authorize` · `Capture` · `Refund` · `GetStatus` · `HandleCallback`.

**Règles propres au module, toutes non négociables :**

| Règle | Pourquoi |
|---|---|
| **Montants en entiers**, plus petite unité, devise explicite | Un `float` sur de l'argent finit par perdre un centime, puis la confiance |
| **Clé d'idempotence obligatoire** sur toute opération débitrice | Un rejeu réseau ne doit jamais débiter deux fois — et le mobile rejoue |
| **Registre en ajout seul** (`UPDATE`/`DELETE` révoqués en SQL) | Un registre réinscriptible ne prouve rien |
| **Signature de rappel vérifiée** avant toute lecture du corps | Un rappel non signé permet à quiconque de déclarer un paiement réussi |
| **Machine à états explicite** : `pending → authorized → captured → refunded` \| `failed` | Un état implicite produit des transitions impossibles à auditer |
| **Aucune donnée de carte ne transite** par le socle | Périmètre PCI-DSS : on manipule des jetons du prestataire, jamais un PAN |
| **Réconciliation** : port de rapprochement entre notre registre et celui du prestataire | Deux systèmes divergent toujours ; sans rapprochement, on l'apprend par le client |
| Le pilote `log` **refuse de démarrer** hors développement | Un paiement silencieusement ignoré en production est le pire scénario possible |

---

## Ordre d'implémentation proposé

Par valeur décroissante rapportée au risque, pas par ordre alphabétique.

1. **Conversion des briques existantes en modules à pilotes** — c'est ce qui rend le socle
   démarrable sans base, et c'est un refactoring sur du code déjà écrit.
2. **`auth`** — sans lui, le socle n'est pas utilisable en production. Exige un arbitrage préalable
   (voir issue de conception).
3. **`notification`** — dépendance de `auth` (vérification d'adresse, TOTP, réinitialisation).
4. **`i18n`** — dépendance de `notification` (gabarits localisés) et de toutes les surfaces.
5. **`payment`** — après `auth` et `audit`, jamais avant : un encaissement sans identité prouvée ni
   journal inaltérable n'est pas défendable.
