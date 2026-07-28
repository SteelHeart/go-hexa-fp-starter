# ADR 011 — Isolation des données : un schéma et un rôle SQL par module

- **Statut** : Accepté
- **Date** : 2026-07-25
- **Remplace** : —
- **Lié à** : [ADR 006](006-outbox-transactionnel.md) · [ADR 009](009-strategie-d-acces-aux-donnees.md) ·
  [ADR 012](012-anatomie-d-un-module-et-pilotes.md) · [`rules/donnees-et-migrations.md`](../../rules/donnees-et-migrations.md)

## Contexte

L'architecture interdit à un module d'en importer un autre : `arch-go` le vérifie, et douze règles
de dépendance échouent la CI si quelqu'un essaie. Mais **cette garde s'arrête au compilateur**.

Rien, dans le code Go, n'empêche l'adaptateur secondaire du module `billing` d'écrire
`SELECT email FROM users`. La requête compile, les tests passent, `arch-go` ne voit rien : il n'y a
aucun import à détecter. La frontière modulaire est parfaitement étanche en Go et parfaitement
poreuse en SQL.

C'est le pire des deux mondes : on croit la contrainte tenue parce qu'un outil est vert, alors que
le contournement le plus probable — deux modules qui partagent une table « pour aller vite » — n'est
surveillé par personne. Et il se découvre le jour où l'on veut extraire un module : il ne part pas,
parce que trois autres lisent ses tables.

Le problème a deux faces distinctes qu'il ne faut pas confondre :

1. **Isolation entre modules** — le module A ne doit pas atteindre les tables du module B.
2. **Isolation entre clients** (multi-tenant) — le client X ne doit pas atteindre les lignes du
   client Y, à l'intérieur d'une même table.

La seconde ne remplace pas la première.

## Décision

### 1. Un schéma PostgreSQL par module, et le nom du schéma est le nom du module

| Provenance | Schéma | Exemple |
|---|---|---|
| Modules **noyau** (`internal/core/`) | **`platform`**, partagé | `platform.outbox_messages` |
| Modules **métier** (`internal/modules/`) | **un schéma par module** | `user_registration.users` |

Les modules noyau partagent `platform` **délibérément** : ce sont des mécanismes techniques du
socle, pas des domaines métier. L'outbox n'a pas de frontière métier à défendre — elle *est* la
frontière. Leur donner six schémas ajouterait de la cérémonie sans ajouter d'isolation.

### 2. Un rôle SQL par module, et le rôle est la garde

Créer les schémas ne suffit pas : sans privilèges distincts, `billing` peut toujours lire
`user_registration.users`. La garde réelle, c'est le rôle.

```
hexa_owner        possède les schémas — DDL. JAMAIS utilisé par l'application.
hexa_migrator     rôle de connexion des MIGRATIONS. Endosse hexa_owner, ne l'hérite pas.
hexa_app          rôle de connexion de l'application. Aucun privilège en propre.
hexa_m_{module}   un par module métier. USAGE sur SON schéma, DML sur SES tables.
hexa_platform     lecture/écriture sur le schéma platform, accordé à tous les modules.
```

**`hexa_migrator` est `NOINHERIT`, exactement comme `hexa_app`**, et ce n'est pas une symétrie
décorative. En `INHERIT`, il porterait passivement le privilège `CREATE` de `hexa_owner` sur tous les
schémas — ce que le §4 interdit, et que `verify.sql` signale. Il l'ENDOSSE à l'ouverture de session
(`ALTER ROLE hexa_migrator SET ROLE hexa_owner`), ce qui laisse `session_user = hexa_migrator` dans
les journaux et `current_user = hexa_owner` sur les objets créés.

Ce rôle était **nommé par `.env.example` et créé par personne** : le dépôt documentait, en
commentaire de `provision.sql`, deux commandes dont le résultat était refusé par son propre garde.
Corrigé en [#84](https://github.com/SteelHeart/go-hexa-fp-starter/issues/84) ; le garde n'a pas
bougé d'une ligne, c'est la provision qui s'y est conformée.

`hexa_app` est **membre** de chaque `hexa_m_{module}` mais n'hérite d'aucun privilège
(`NOINHERIT`). Un adaptateur secondaire prend son rôle pour la durée de sa transaction :

```sql
SET LOCAL ROLE hexa_m_user_registration;
```

`SET LOCAL` est borné à la transaction : au `COMMIT`, la connexion retourne au pool sans état
résiduel. Une requête du module `billing` visant `user_registration.users` échoue alors sur
`permission denied` — **au moment où elle est écrite, pas le jour où l'on veut extraire le module**.

C'est ce qui transforme une convention en contrainte.

### 3. RLS par défaut sur toute table portant une donnée de client

Toute table d'un schéma de module métier :

```sql
ALTER TABLE m.t ENABLE ROW LEVEL SECURITY;
ALTER TABLE m.t FORCE ROW LEVEL SECURITY;   -- s'applique même au propriétaire
CREATE POLICY tenant_isolation ON m.t
    USING (tenant_id = current_setting('app.current_tenant', true));
```

`FORCE` compte autant que `ENABLE` : sans lui, le propriétaire de la table contourne la politique,
et le propriétaire est justement le rôle qu'on utilise pendant un incident, à trois heures du matin,
pour « juste vérifier une chose ».

Le tenant vient de `set_config('app.current_tenant', …, true)`, posé par `RunInTx` depuis le
contexte. **Absence de réglage = aucune ligne visible**, pas « toutes les lignes » : deny par défaut
jusque dans la politique.

### 4. Le rôle applicatif ne possède rien

Repris de [`rules/donnees-et-migrations.md`](../../rules/donnees-et-migrations.md) §6 et rendu
exécutable ici : `hexa_app` et les rôles de module n'ont ni `CREATE`, ni `ALTER`, ni `DROP`. Une
injection SQL réussie peut au pire lire et écrire ce que le module lit et écrit déjà — elle ne peut
pas créer une table, ni **désactiver une politique RLS**.

### 5. Le journal d'audit est en ajout seul, par privilège

`UPDATE` et `DELETE` sont **révoqués** sur `platform.audit_log` pour tous les rôles applicatifs. Un
journal qu'on peut réécrire ne prouve rien ; le documenter ne suffit pas, il faut que la base refuse.

### 6. Les rôles sont provisionnés, pas migrés

Un rôle est un objet de **cluster**, partagé par toutes les bases ; une migration agit sur **une**
base. Surtout, `CREATE ROLE` exige des privilèges que le rôle de migration ne doit pas posséder — la
décision §4 serait vide si la migration pouvait se créer des rôles.

Donc : [`deploy/postgres/provision.sql`](../../deploy/postgres/provision.sql), exécuté **une fois**
par un administrateur, hors goose. Les migrations supposent les rôles présents, et un `GRANT` vers
un rôle absent échoue bruyamment — ce qui est le comportement voulu.

## Conséquences

**Ce qu'on gagne**

- La frontière modulaire est vérifiée par la base, là où `arch-go` ne peut rien voir.
- Extraire un module en service séparé devient mécanique : son schéma part avec lui.
- Une injection SQL est bornée au périmètre du module touché.
- Le journal d'audit est inaltérable par construction.

**Ce qu'on paie**

- Un adaptateur secondaire doit poser son `SET LOCAL ROLE`. Oublié, il tombe sur `permission
  denied` — bruyant, donc acceptable ; c'est l'inverse d'un échec silencieux.
- Les jointures entre modules deviennent **impossibles en SQL**. C'est le but, pas un effet de bord :
  une jointure inter-modules est exactement le couplage qu'on refuse. La donnée d'un autre module
  s'obtient par son port, ou par un événement.
- Le provisionnement devient un prérequis documenté du déploiement.

**Ce que cette décision NE tranche pas**

- **Le modèle de multi-tenant lui-même** — colonne `tenant_id`, base par client, ou schéma par
  client. La §3 dit *comment* isoler une fois le modèle choisi, pas lequel choisir. Le module
  `tenancy` n'existe pas ; tant qu'il n'existe pas, aucune table ne porte de `tenant_id` et aucune
  politique RLS n'est écrite. **En écrire une maintenant serait décorer une décision non prise.**
- **La base de données.** `postgres` reste un pilote parmi d'autres (ADR 012). Cet ADR décrit
  l'isolation *sous PostgreSQL* ; un pilote SQLite (issue #36) devra dire ce qu'il ne peut pas
  garantir — il n'a ni rôle, ni RLS, ni schéma.

## Alternatives écartées

| Alternative | Pourquoi non |
|---|---|
| **Une base par module** | Tue l'outbox transactionnel : l'écriture métier et l'événement ne seraient plus dans la même transaction (ADR 006). Le prix est trop élevé pour ce qu'on gagne |
| **Un seul schéma, discipline humaine** | C'est l'état actuel, et c'est ce qui a produit le problème. Une règle non outillée n'existe pas |
| **Préfixe de table (`ur_users`)** | Cosmétique : aucun privilège ne s'y attache, donc aucune garde. Renomme le problème |
| **Vérifier les requêtes par analyse statique** | Le SQL est une chaîne de caractères ; l'analyse rendrait des faux négatifs sur toute requête composée. Le moteur, lui, ne se trompe pas |
| **RLS seule, sans rôle par module** | Résout l'isolation entre clients, pas entre modules. Les deux problèmes sont distincts |
