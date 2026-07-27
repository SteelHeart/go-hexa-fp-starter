-- Provisionnement des rôles PostgreSQL du socle.
--
-- ┌─ À LIRE AVANT D'EXÉCUTER ──────────────────────────────────────────────────┐
-- │ Ce fichier n'est PAS une migration et n'est PAS exécuté par goose.         │
-- │                                                                            │
-- │ Un rôle est un objet de CLUSTER, partagé par toutes les bases ; une        │
-- │ migration agit sur UNE base. Surtout, `CREATE ROLE` exige des privilèges   │
-- │ que le rôle de migration ne doit pas posséder — sinon la décision « le     │
-- │ rôle applicatif ne possède rien » (ADR 011 §4) serait vide, puisqu'une     │
-- │ migration pourrait se fabriquer les droits qui lui manquent.               │
-- │                                                                            │
-- │ Exécution : UNE FOIS, par un administrateur, avant la première migration.  │
-- │                                                                            │
-- │   psql "$DB_SUPERUSER_DSN" -v ON_ERROR_STOP=1 -f deploy/postgres/provision.sql
-- │                                                                            │
-- │ Idempotent : relançable sans effet de bord.                                │
-- └────────────────────────────────────────────────────────────────────────────┘
--
-- AUCUN MOT DE PASSE ICI. Les rôles sont créés sans secret ; celui de `hexa_app`
-- est posé hors dépôt, par le mécanisme de secrets de l'environnement
-- (rules/securite.md : aucun secret versionné, même en exemple).
--
--   ALTER ROLE hexa_app WITH PASSWORD '…';   -- hors dépôt, jamais ici
--
-- Voir documentation/adr/011-isolation-des-donnees-par-module.md

\set ON_ERROR_STOP on

BEGIN;

-- ── hexa_owner : propriétaire des schémas ────────────────────────────────────
--
-- C'est le rôle sous lequel s'exécutent les MIGRATIONS. Il possède tout et ne
-- sert jamais à l'application.
--
-- NOLOGIN volontairement : on ne se connecte pas « en tant que hexa_owner », on
-- l'ENDOSSE. Le rôle de connexion de DB_MIGRATION_DSN doit être membre de
-- hexa_owner, et chaque migration fait `SET LOCAL ROLE hexa_owner`. Un rôle
-- propriétaire sans mot de passe est un rôle qu'aucune fuite de secret n'ouvre.
--
--   CREATE ROLE hexa_migrator LOGIN;          -- hors dépôt
--   GRANT hexa_owner TO hexa_migrator;        -- hors dépôt
--
-- En CI, l'administrateur de la base tient ce rôle : un superutilisateur peut
-- toujours endosser n'importe quel rôle.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'hexa_owner') THEN
        CREATE ROLE hexa_owner NOLOGIN;
    END IF;
END
$$;

-- « Propriétaire des schémas » ne suffit pas à en CRÉER un : `CREATE SCHEMA`
-- exige le privilège CREATE sur la BASE, qui n'appartient à personne d'autre que
-- son propriétaire. Sans la ligne ci-dessous, la toute première migration échoue
-- sur un message qui ne nomme aucun schéma :
--
--   ERROR: permission denied for database hexa
--
-- Le privilège porte sur la base à laquelle CE script est connecté. C'est la
-- raison pour laquelle DB_SUPERUSER_DSN désigne la base APPLICATIVE et non
-- `postgres` : les rôles sont des objets de cluster, mais ce droit-ci ne l'est pas.
DO $$
BEGIN
    EXECUTE format('GRANT CREATE, CONNECT ON DATABASE %I TO hexa_owner', current_database());
END
$$;

-- ── hexa_platform : accès aux tables des modules NOYAU ───────────────────────
--
-- Accordé à tous les modules métier : l'outbox, l'idempotence et l'audit sont des
-- mécanismes du socle, pas le domaine réservé de quelqu'un.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'hexa_platform') THEN
        CREATE ROLE hexa_platform NOLOGIN;
    END IF;
END
$$;

-- ── hexa_app : le rôle de CONNEXION de l'application ─────────────────────────
--
-- NOINHERIT est le cœur du dispositif, pas un détail de configuration.
--
-- Avec INHERIT, `hexa_app` cumulerait passivement les privilèges de TOUS les
-- rôles de module dont il est membre : chaque requête pourrait alors atteindre
-- n'importe quelle table, et l'isolation par schéma ne serait plus qu'un rangement.
--
-- Avec NOINHERIT, il ne peut rien par lui-même. Un adaptateur secondaire doit
-- prendre explicitement le rôle de son module — `SET LOCAL ROLE hexa_m_…` — pour
-- la durée de sa transaction. C'est ce qui rend la frontière effective.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'hexa_app') THEN
        CREATE ROLE hexa_app LOGIN NOINHERIT;
    ELSE
        ALTER ROLE hexa_app NOINHERIT;
    END IF;
END
$$;

GRANT hexa_platform TO hexa_app;

-- ── Un rôle par module MÉTIER ────────────────────────────────────────────────
--
-- Cette liste grandit d'une ligne par module métier persistant. Aujourd'hui elle
-- est VIDE : `user_registration` n'a pas encore d'adaptateur secondaire, donc pas
-- de table. On ne provisionne pas un rôle pour des données qui n'existent pas.
--
-- Gabarit, à décommenter le jour où le module a réellement un schéma :
--
--   DO $$
--   BEGIN
--       IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'hexa_m_user_registration') THEN
--           CREATE ROLE hexa_m_user_registration NOLOGIN;
--       END IF;
--   END
--   $$;
--   GRANT hexa_m_user_registration TO hexa_app;

COMMIT;

-- ── Vérification ─────────────────────────────────────────────────────────────
--
-- Affiche l'état obtenu. `rolinherit` DOIT être false pour hexa_app.
SELECT rolname,
       rolcanlogin AS peut_se_connecter,
       rolinherit  AS herite_des_privileges,
       rolsuper    AS superutilisateur
FROM pg_roles
WHERE rolname LIKE 'hexa\_%'
ORDER BY rolname;
