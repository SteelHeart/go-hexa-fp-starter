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
-- AUCUN MOT DE PASSE ICI. Les rôles de connexion — `hexa_app` et `hexa_migrator`
-- — sont créés sans secret ; le leur est posé hors dépôt, par le mécanisme de
-- secrets de l'environnement (rules/securite.md : aucun secret versionné, même
-- en exemple).
--
--   ALTER ROLE hexa_app WITH PASSWORD '…';   -- hors dépôt, jamais ici
--
-- Un rôle LOGIN sans mot de passe ne se connecte pas : ce n'est donc pas un
-- trou, c'est un état inachevé qui échoue bruyamment. En DÉVELOPPEMENT,
-- `task db:credentials` lit `.env` — non versionné — et pose ces mots de passe ;
-- il REFUSE de s'exécuter hors `development` et `test`.
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
-- l'ENDOSSE. Le rôle de connexion de DB_MIGRATION_DSN est `hexa_migrator`,
-- provisionné plus bas, et chaque migration fait `SET LOCAL ROLE hexa_owner`.
-- Un rôle propriétaire sans mot de passe est un rôle qu'aucune fuite de secret
-- n'ouvre.
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

-- Depuis PostgreSQL 15, le schéma `public` n'accorde plus CREATE à PUBLIC, et il
-- appartient à `pg_database_owner`. Sans la ligne ci-dessous, goose échoue avant
-- la première migration, sur la création de sa propre table de suivi :
--
--   ERROR: permission denied for schema public
--
-- Mesuré, pas déduit. C'est le quatrième maillon de la cascade de l'issue #84 :
-- chacun était masqué par le précédent, donc invisible tant que le premier
-- n'était pas corrigé.
GRANT CREATE ON SCHEMA public TO hexa_owner;

-- ── hexa_migrator : le rôle de CONNEXION des migrations ──────────────────────
--
-- Ce rôle n'existait nulle part. `.env.example` le désignait, `provision.sql` ne
-- le créait pas, et les deux lignes que ce fichier donnait en exemple —
-- `CREATE ROLE hexa_migrator LOGIN` puis `GRANT hexa_owner TO hexa_migrator` —
-- produisaient un état que `verify.sql` REFUSE. Le dépôt documentait une
-- configuration que son propre garde rejetait (issue #84).
--
-- NOINHERIT, exactement pour la même raison que `hexa_app`, et c'est ce qui
-- réconcilie le garde et la documentation : avec INHERIT, `hexa_migrator`
-- porterait passivement le privilège CREATE de `hexa_owner` sur tous les
-- schémas, et `verify.sql` §2 le signalerait — à raison.
--
--   ERROR: privilège CREATE accordé à un rôle applicatif — hexa_migrator …
--
-- `has_schema_privilege` respecte l'attribut d'héritage : NOINHERIT, le
-- privilège n'est PAS tenu, il est seulement ENDOSSABLE. Le garde reste strict,
-- et aucune exception ne lui est ajoutée — c'est le point.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'hexa_migrator') THEN
        CREATE ROLE hexa_migrator LOGIN NOINHERIT;
    ELSE
        ALTER ROLE hexa_migrator NOINHERIT;
    END IF;
END
$$;

GRANT hexa_owner TO hexa_migrator;

-- Endossement AUTOMATIQUE à l'ouverture de session.
--
-- Sans cette ligne, il faudrait un `SET ROLE hexa_owner` avant chaque commande —
-- ce que les migrations font déjà, mais pas goose pour sa propre table de suivi,
-- ni `psql` pour une commande ponctuelle. Un dispositif qui exige de se souvenir
-- d'une commande sera oublié.
--
-- L'effet est exactement celui voulu :
--
--   session_user  = hexa_migrator   ← ce que voient les journaux et pg_stat
--   current_user  = hexa_owner      ← ce qui possède les objets créés
--
-- Et `RESET ROLE` en fin de migration retombe sur ce défaut de session, pas sur
-- `session_user` : goose inscrit donc sa version en tant que propriétaire.
-- Vérifié sur un cluster réel avant d'être écrit ici.
ALTER ROLE hexa_migrator SET ROLE hexa_owner;

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
