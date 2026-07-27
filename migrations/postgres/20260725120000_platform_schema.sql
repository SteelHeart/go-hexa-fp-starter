-- Schéma `platform` : les tables des modules NOYAU.
--
-- Quatre modules noyau persistent : outbox, idempotency, audit, dynconf. Le
-- cinquième pilote Postgres — celui de l'ordonnanceur — n'a aucune table : il
-- utilise les verrous consultatifs, qui ne créent rien. Le module storage n'a pas
-- de pilote Postgres du tout.
--
-- Chaque colonne ci-dessous est dérivée d'une requête EXISTANTE, dans
-- internal/core/{module}/drivers/postgres/postgres.go. Aucune n'est anticipée :
-- une colonne que personne n'écrit est une dette, pas une préparation.
--
-- Prérequis : deploy/postgres/provision.sql a été exécuté. Un GRANT vers un rôle
-- absent échoue ici, et c'est voulu — deny par défaut.
--
-- Voir ADR 011 et rules/donnees-et-migrations.md.

-- +goose Up
-- +goose StatementBegin

-- Tout ce qui suit est créé PAR hexa_owner, et pas seulement dans son schéma.
--
-- Sans cette ligne, le propriétaire des tables serait le rôle qui exécute la
-- migration — qui peut être un administrateur en CI. Deux conséquences, aucune
-- visible immédiatement : le `ALTER DEFAULT PRIVILEGES FOR ROLE hexa_owner` plus
-- bas ne s'appliquerait à rien, et la propriété des tables varierait d'un
-- environnement à l'autre.
--
-- SET LOCAL : borné à la transaction de goose, donc aucun état résiduel APRÈS
-- elle. Mais goose écrit sa propre ligne de version DANS cette transaction :
-- il faut donc relâcher le rôle avant la fin du bloc — voir le RESET ROLE final.
--
-- Prérequis : le rôle de DB_MIGRATION_DSN est membre de hexa_owner.
SET LOCAL ROLE hexa_owner;

-- ─────────────────────────────────────────────────────────────────────────────
-- Schéma
-- ─────────────────────────────────────────────────────────────────────────────
--
-- AUTHORIZATION hexa_owner : le rôle applicatif ne possède pas le schéma, donc
-- il ne peut ni le modifier, ni le supprimer, ni désactiver une politique posée
-- dessus (ADR 011 §4).
CREATE SCHEMA IF NOT EXISTS platform AUTHORIZATION hexa_owner;

COMMENT ON SCHEMA platform IS
    'Tables des modules noyau. Partagé délibérément : ce sont des mécanismes du socle, pas des domaines métier (ADR 011).';

-- ─────────────────────────────────────────────────────────────────────────────
-- platform.outbox_messages — ADR 006
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS platform.outbox_messages (
    -- UUID v7 fourni par le pilote, PAS un défaut de la base : le domaine de
    -- l'outbox n'importe aucun générateur d'identifiants, c'est ce qui le rend
    -- testable sans rien. Un DEFAULT gen_random_uuid() produirait en prime un v4
    -- aléatoire, qui fragmente l'index (rules §7).
    id            uuid        PRIMARY KEY,

    event_type    text        NOT NULL CHECK (event_type <> ''),
    aggregate_id  text        NOT NULL,

    -- jsonb et non text : le contenu est du JSON opaque, mais le stocker en jsonb
    -- le fait VALIDER à l'écriture. Un événement au JSON cassé serait sinon
    -- découvert par le consommateur, après publication, quand il est trop tard
    -- pour refuser la transaction qui l'a produit.
    payload       jsonb       NOT NULL,

    trace_parent  text        NOT NULL DEFAULT '',
    headers       jsonb       NOT NULL DEFAULT '{}'::jsonb,

    -- text + CHECK plutôt qu'un type ENUM : l'ALTER d'un ENUM est pénible et non
    -- transactionnel selon les versions (rules §7).
    status        text        NOT NULL DEFAULT 'pending'
                              CHECK (status IN ('pending', 'done', 'failed')),
    attempts      integer     NOT NULL DEFAULT 0 CHECK (attempts >= 0),

    created_at    timestamptz NOT NULL DEFAULT now(),
    available_at  timestamptz NOT NULL DEFAULT now(),
    processed_at  timestamptz,
    last_error    text
);

COMMENT ON TABLE platform.outbox_messages IS
    'Événements en attente de publication. Un message failed n''est JAMAIS supprimé : c''est la seule trace de ce qui n''a pas été publié.';

-- Index PARTIEL, et c'est le point important.
--
-- Il indexe uniquement les lignes `pending`, c'est-à-dire la file d'attente. Les
-- messages `done` — qui finissent par représenter la quasi-totalité de la table —
-- n'y entrent jamais. L'index reste donc de la taille du RETARD, pas de celle de
-- l'historique, et il ne grossit pas avec le trafic écoulé.
--
-- CREATE INDEX simple et non CONCURRENTLY : la table vient d'être créée, elle est
-- vide. La règle §5 exige CONCURRENTLY sur une GROSSE table existante.
CREATE INDEX IF NOT EXISTS outbox_messages_pending_idx
    ON platform.outbox_messages (available_at)
    WHERE status = 'pending';

-- ─────────────────────────────────────────────────────────────────────────────
-- platform.idempotency_keys
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS platform.idempotency_keys (
    -- La clé primaire EST la garde d'exclusivité : c'est la contrainte d'unicité
    -- qui tranche entre deux requêtes concurrentes, pas un verrou applicatif.
    key          text        PRIMARY KEY CHECK (key <> ''),

    -- Empreinte de la charge utile. Une même clé rejouée avec un corps différent
    -- est un CONFLIT, pas un rejeu : c'est cette colonne qui permet de le voir.
    fingerprint  text        NOT NULL,

    status       text        NOT NULL CHECK (status IN ('in_flight', 'done')),

    -- Réponse mémorisée, rendue telle quelle au rejeu. bytea et non jsonb : le
    -- pilote la traite comme opaque, et une réponse n'est pas nécessairement du
    -- JSON.
    response     bytea,

    expires_at   timestamptz NOT NULL,
    completed_at timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now()
);

COMMENT ON TABLE platform.idempotency_keys IS
    'Réservations d''idempotence. Hors transaction métier par conception : une réservation invisible aux autres connexions ne protégerait de rien.';

-- Sert la purge : DELETE ... WHERE expires_at < now().
CREATE INDEX IF NOT EXISTS idempotency_keys_expires_at_idx
    ON platform.idempotency_keys (expires_at);

-- ─────────────────────────────────────────────────────────────────────────────
-- platform.audit_log — ajout seul
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS platform.audit_log (
    -- bigint IDENTITY et non uuid : rien n'expose cet identifiant, et un journal
    -- en ajout seul s'écrit strictement en queue. Une séquence est plus compacte
    -- qu'un UUID et ordonne naturellement. La règle §7 vise les identifiants
    -- d'entités métier, qui circulent ; celui-ci ne sort jamais de la table.
    id          bigint      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    -- Un identifiant, JAMAIS un nom ni une adresse (rules/securite.md §5).
    actor       text        NOT NULL CHECK (actor <> ''),
    -- Le fait, au passé et à la forme métier : "user.registered", pas "INSERT users".
    action      text        NOT NULL CHECK (action <> ''),
    entity_type text        NOT NULL CHECK (entity_type <> ''),
    entity_id   text        NOT NULL CHECK (entity_id <> ''),

    -- Aucune donnée personnelle en clair : le journal est conservé longtemps et
    -- lu par des humains. Le CHECK ne peut pas le vérifier — c'est une revue.
    metadata    jsonb       NOT NULL DEFAULT '{}'::jsonb,

    -- Horodatage INJECTÉ par le module, jamais now() : l'instant vient d'un port,
    -- ce qui rend un test d'audit déterministe. Pas de DEFAULT, donc : un défaut
    -- masquerait un appelant qui a oublié de fournir l'instant.
    occurred_at timestamptz NOT NULL
);

COMMENT ON TABLE platform.audit_log IS
    'Journal d''audit en AJOUT SEUL. UPDATE et DELETE sont révoqués : un journal réinscriptible ne prouve rien (ADR 011 §5).';

CREATE INDEX IF NOT EXISTS audit_log_entity_idx
    ON platform.audit_log (entity_type, entity_id, occurred_at DESC);

CREATE INDEX IF NOT EXISTS audit_log_actor_idx
    ON platform.audit_log (actor, occurred_at DESC);

-- ─────────────────────────────────────────────────────────────────────────────
-- platform.dynamic_config
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS platform.dynamic_config (
    kind       text        NOT NULL CHECK (kind IN ('flag', 'setting')),
    key        text        NOT NULL CHECK (key <> ''),
    -- La valeur PEUT être vide : c'est une valeur légitime. La clé et la nature,
    -- non — d'où les CHECK ci-dessus et leur absence ici.
    value      text        NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now(),

    -- Clé composite : c'est la cible du ON CONFLICT (kind, key) du pilote.
    PRIMARY KEY (kind, key)
);

COMMENT ON TABLE platform.dynamic_config IS
    'Drapeaux et réglages modifiables à chaud. Un drapeau absent vaut ÉTEINT : deny par défaut.';

-- ─────────────────────────────────────────────────────────────────────────────
-- Privilèges — ADR 011 §4 et §5
-- ─────────────────────────────────────────────────────────────────────────────
--
-- hexa_platform reçoit le DML, jamais le DDL. Aucun GRANT ... ON SCHEMA ... CREATE :
-- une injection SQL réussie ne pourra pas créer de table ni altérer une politique.
GRANT USAGE ON SCHEMA platform TO hexa_platform;

GRANT SELECT, INSERT, UPDATE, DELETE
    ON platform.outbox_messages, platform.idempotency_keys, platform.dynamic_config
    TO hexa_platform;

-- L'audit fait exception, et c'est TOUT le sujet : INSERT et SELECT seulement.
-- Pas d'UPDATE, pas de DELETE, pas même pour purger — une rétention se décide et
-- s'exécute avec le rôle propriétaire, jamais avec le rôle applicatif.
GRANT SELECT, INSERT ON platform.audit_log TO hexa_platform;

REVOKE UPDATE, DELETE, TRUNCATE ON platform.audit_log FROM hexa_platform;
REVOKE UPDATE, DELETE, TRUNCATE ON platform.audit_log FROM PUBLIC;

-- La séquence de l'identité doit être utilisable, sinon l'INSERT échoue.
GRANT USAGE ON ALL SEQUENCES IN SCHEMA platform TO hexa_platform;

-- Les privilèges par défaut valent pour les tables CRÉÉES PLUS TARD par
-- hexa_owner : sans cette ligne, chaque nouvelle migration devrait penser à
-- refaire ses GRANT, et celle qui oublierait produirait un « permission denied »
-- en production, pas en revue.
ALTER DEFAULT PRIVILEGES FOR ROLE hexa_owner IN SCHEMA platform
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO hexa_platform;

ALTER DEFAULT PRIVILEGES FOR ROLE hexa_owner IN SCHEMA platform
    GRANT USAGE ON SEQUENCES TO hexa_platform;

-- ─────────────────────────────────────────────────────────────────────────────
-- Row Level Security
-- ─────────────────────────────────────────────────────────────────────────────
--
-- AUCUNE politique n'est posée ici, et c'est une décision, pas un oubli.
--
-- La RLS isole des CLIENTS entre eux (ADR 011 §3). Aucune de ces quatre tables ne
-- porte de donnée de client : ce sont des mécanismes du socle — une file de
-- publication, des réservations, un journal, des drapeaux. Il n'y a rien à
-- cloisonner tant que le modèle de multi-tenant n'est pas tranché, et le module
-- `tenancy` n'existe pas.
--
-- Écrire une politique sur une colonne `tenant_id` que personne ne remplit
-- donnerait l'apparence d'une garde sans la garde. C'est exactement le genre de
-- décor que ce dépôt refuse.
--
-- Le jour où une table porte un tenant, la forme est fixée par l'ADR 011 §3, et
-- deploy/postgres/verify.sql refuse toute table de schéma métier sans RLS.

-- ─────────────────────────────────────────────────────────────────────────────
-- Relâchement du rôle — OBLIGATOIRE en dernière instruction
-- ─────────────────────────────────────────────────────────────────────────────
--
-- Sans cette ligne, la migration échoue APRÈS avoir tout créé :
--
--   failed to insert new goose version:
--   ERROR: permission denied for table goose_db_version
--
-- Parce que goose enregistre la version dans la MÊME transaction que la
-- migration. `SET LOCAL` est borné à la transaction, pas à la migration : la
-- table de versions appartient au rôle de connexion, et hexa_owner n'y a aucun
-- droit. La transaction est alors annulée en entier — donc rien n'est appliqué,
-- et le message accuse une table qui n'a rien à voir avec le schéma migré.
--
-- Toute migration future qui prend un rôle doit le relâcher ici, pour la même
-- raison. C'est la contrepartie du SET LOCAL ROLE en tête, pas une précaution.
RESET ROLE;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

SET LOCAL ROLE hexa_owner;

-- DROP SCHEMA ... CASCADE emporte tables, index, séquences et privilèges.
--
-- Ce Down est DESTRUCTEUR et il est réellement testé en CI sur une base jetable
-- (rules §5 : un Down faux est pire que pas de Down). Sur une base qui porte des
-- données, il détruit le journal d'audit et la file de publication.
--
-- Il n'existe que pour le cycle de vie d'une base de test. En production, un
-- retour arrière se fait par une migration AVANT, jamais par ce Down.
DROP SCHEMA IF EXISTS platform CASCADE;

-- Même raison que dans le Up : goose met à jour sa table de versions dans cette
-- transaction, sous le rôle courant.
RESET ROLE;

-- +goose StatementEnd
