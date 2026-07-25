-- Vérification des invariants d'isolation — ADR 011.
--
-- Une règle non outillée n'existe pas. Les décisions de l'ADR 011 sont des
-- propriétés de la BASE : elles ne se vérifient ni par arch-go, ni par un test Go,
-- ni par une relecture. Ce script les interroge et ÉCHOUE si l'une est fausse.
--
-- Exécution — après les migrations, avec un rôle qui peut lire les catalogues :
--
--   psql "$DB_MIGRATION_DSN" -v ON_ERROR_STOP=1 -f migrations/postgres/verify.sql
--
-- Sortie : silencieuse si tout tient, `ERROR` et code de retour non nul sinon.
-- ⚠️ Vérifier le CODE DE RETOUR, pas seulement la sortie : une commande qui n'a
-- pas tourné rend une sortie vide, ce qui ressemble à « propre ».

\set ON_ERROR_STOP on

-- ─────────────────────────────────────────────────────────────────────────────
-- 1. hexa_app n'hérite d'aucun privilège
-- ─────────────────────────────────────────────────────────────────────────────
--
-- Si quelqu'un repasse ce rôle en INHERIT, toute l'isolation par rôle de module
-- s'effondre en silence : chaque requête atteindrait toutes les tables, et rien
-- n'échouerait. C'est la régression la plus grave et la plus discrète du lot.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'hexa_app') THEN
        RAISE EXCEPTION 'ADR 011 : le rôle hexa_app est absent — deploy/postgres/provision.sql n''a pas été exécuté';
    END IF;

    IF (SELECT rolinherit FROM pg_roles WHERE rolname = 'hexa_app') THEN
        RAISE EXCEPTION
            'ADR 011 §2 : hexa_app est INHERIT — il cumule les privilèges de tous les modules, l''isolation par schéma ne vaut plus rien';
    END IF;

    IF (SELECT rolsuper FROM pg_roles WHERE rolname = 'hexa_app') THEN
        RAISE EXCEPTION 'ADR 011 §4 : hexa_app est superutilisateur';
    END IF;
END
$$;

-- ─────────────────────────────────────────────────────────────────────────────
-- 2. Aucun rôle applicatif ne peut faire de DDL
-- ─────────────────────────────────────────────────────────────────────────────
--
-- CREATE sur un schéma suffit à créer une table, donc à contourner tout le reste.
DO $$
DECLARE
    fautif text;
BEGIN
    SELECT string_agg(format('%s sur le schéma %s', r.rolname, n.nspname), ', ')
      INTO fautif
      FROM pg_namespace n
      CROSS JOIN pg_roles r
     WHERE r.rolname LIKE 'hexa\_%'
       AND r.rolname <> 'hexa_owner'
       AND n.nspname NOT LIKE 'pg\_%'
       AND n.nspname <> 'information_schema'
       AND has_schema_privilege(r.rolname, n.nspname, 'CREATE');

    IF fautif IS NOT NULL THEN
        RAISE EXCEPTION 'ADR 011 §4 : privilège CREATE accordé à un rôle applicatif — %', fautif;
    END IF;
END
$$;

-- ─────────────────────────────────────────────────────────────────────────────
-- 3. Le journal d'audit est en ajout seul
-- ─────────────────────────────────────────────────────────────────────────────
DO $$
DECLARE
    faute text;
BEGIN
    SELECT string_agg(format('%s peut %s', r.rolname, p.priv), ', ')
      INTO faute
      FROM pg_roles r
      CROSS JOIN (VALUES ('UPDATE'), ('DELETE'), ('TRUNCATE')) AS p(priv)
     WHERE r.rolname LIKE 'hexa\_%'
       AND r.rolname <> 'hexa_owner'
       AND has_table_privilege(r.rolname, 'platform.audit_log', p.priv);

    IF faute IS NOT NULL THEN
        RAISE EXCEPTION
            'ADR 011 §5 : platform.audit_log n''est plus en ajout seul — %', faute;
    END IF;
END
$$;

-- ─────────────────────────────────────────────────────────────────────────────
-- 4. RLS par défaut sur toute table de schéma MÉTIER
-- ─────────────────────────────────────────────────────────────────────────────
--
-- L'exemption de `platform` est NOMMÉE, jamais silencieuse : ses tables ne portent
-- aucune donnée de client (voir la migration du schéma). Une exemption écrite se
-- rediscute ; une exemption implicite se découvre après la fuite.
--
-- `relforcerowsecurity` compte autant que `relrowsecurity` : sans FORCE, le
-- propriétaire de la table contourne la politique — et c'est le rôle qu'on
-- utilise pendant un incident.
DO $$
DECLARE
    sans_rls text;
BEGIN
    SELECT string_agg(format('%s.%s', n.nspname, c.relname), ', ' ORDER BY n.nspname, c.relname)
      INTO sans_rls
      FROM pg_class c
      JOIN pg_namespace n ON n.oid = c.relnamespace
     WHERE c.relkind = 'r'
       AND n.nspname NOT IN ('platform', 'public', 'information_schema')
       AND n.nspname NOT LIKE 'pg\_%'
       AND pg_get_userbyid(n.nspowner) = 'hexa_owner'
       AND NOT (c.relrowsecurity AND c.relforcerowsecurity);

    IF sans_rls IS NOT NULL THEN
        RAISE EXCEPTION
            'ADR 011 §3 : table de module métier sans RLS activée ET forcée — %', sans_rls;
    END IF;
END
$$;

-- ─────────────────────────────────────────────────────────────────────────────
-- 5. Aucun module ne voit le schéma d'un autre module
-- ─────────────────────────────────────────────────────────────────────────────
--
-- La garde centrale de l'ADR 011. Un rôle `hexa_m_a` qui obtient USAGE sur le
-- schéma du module B rouvre la porte que tout ce dispositif ferme — et il
-- l'obtient d'un GRANT ajouté « juste pour une jointure », en général un vendredi.
DO $$
DECLARE
    fuite text;
BEGIN
    SELECT string_agg(format('%s voit le schéma %s', r.rolname, n.nspname), ', ')
      INTO fuite
      FROM pg_namespace n
      CROSS JOIN pg_roles r
     WHERE r.rolname LIKE 'hexa\_m\_%'
       AND n.nspname NOT IN ('platform', 'public', 'information_schema')
       AND n.nspname NOT LIKE 'pg\_%'
       AND pg_get_userbyid(n.nspowner) = 'hexa_owner'
       -- Le rôle du module s'appelle hexa_m_{schéma} : tout accès à un AUTRE
       -- schéma est une violation.
       AND r.rolname <> 'hexa_m_' || n.nspname
       AND has_schema_privilege(r.rolname, n.nspname, 'USAGE');

    IF fuite IS NOT NULL THEN
        RAISE EXCEPTION
            'ADR 011 §2 : un module atteint le schéma d''un autre module — %', fuite;
    END IF;
END
$$;

\echo 'ADR 011 : tous les invariants d''isolation sont vérifiés.'
