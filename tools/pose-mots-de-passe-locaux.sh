#!/usr/bin/env sh
#
# Pose les mots de passe des rôles de connexion — DÉVELOPPEMENT SEULEMENT (#84).
#
#   ./tools/pose-mots-de-passe-locaux.sh
#   ./tools/pose-mots-de-passe-locaux.sh --temoin
#
# ─────────────────────────────────────────────────────────────────────────────
# Pourquoi cette étape existe
# ─────────────────────────────────────────────────────────────────────────────
# `deploy/postgres/provision.sql` crée `hexa_app` et `hexa_migrator` SANS mot de
# passe, et c'est une décision : un rôle LOGIN dont le secret serait versionné
# ouvrirait la base à quiconque lit le dépôt.
#
# Mais un rôle LOGIN sans mot de passe ne se connecte pas. Sur un clone neuf,
# `task up` échouait donc juste après le provisionnement, sur un message qui ne
# nommait pas la cause :
#
#   goose: failed to connect: password authentication failed for user "hexa_migrator"
#
# Il manquait l'étape qui relie les deux : celle qui prend les secrets LOCAUX,
# déjà présents dans `.env` — non versionné —, et les pose sur le cluster local.
#
# ─────────────────────────────────────────────────────────────────────────────
# La source unique est `.env`, jamais ce fichier
# ─────────────────────────────────────────────────────────────────────────────
# Les mots de passe ne sont pas écrits ici : ils sont EXTRAITS de `DB_DSN` et
# `DB_MIGRATION_DSN`. Le cluster et l'application lisent donc la même valeur par
# construction, et il n'existe aucun second endroit où elle pourrait diverger.
#
# C'est la faute que ce dépôt a déjà payée avec les trois seuils de couverture
# écrits à trois endroits, dont un mort.
#
# ─────────────────────────────────────────────────────────────────────────────
# Le refus hors développement
# ─────────────────────────────────────────────────────────────────────────────
# En UAT et en production, les secrets viennent du mécanisme de l'environnement,
# jamais d'un fichier lu par un script du dépôt. Ce script REFUSE donc de
# s'exécuter si `APP_ENV` n'est pas explicitement `development` ou `test` —
# valeur absente comprise, parce qu'un défaut permissif se laisse toujours
# atteindre par accident.
#
# ⚠️ Ce refus est livré avec le cas qui l'exerce (ADR 013) : `--temoin` relance
#    ce même script avec `APP_ENV=production` et échoue si le refus n'a pas lieu.

set -eu

usage() {
    echo "usage: $0 [--temoin]" >&2
    exit 2
}

# ── Témoin : le refus doit se CONSTATER, pas se supposer ─────────────────────
#
# Un garde qui ne refuse rien ressemble en tout point à un garde satisfait. La
# seule façon de savoir que celui-ci refuse est de le lui faire faire.
temoin() {
    echo "témoin : APP_ENV=production doit faire refuser ce script"
    if APP_ENV=production \
       DB_SUPERUSER_DSN=inutilise DB_DSN=inutilise DB_MIGRATION_DSN=inutilise \
       "$0" >/dev/null 2>&1; then
        echo "  ÉCHEC : le script a accepté de s'exécuter en production" >&2
        return 1
    fi
    echo "  refus confirmé"

    echo "témoin : APP_ENV absent doit faire refuser ce script"
    if env -u APP_ENV \
       DB_SUPERUSER_DSN=inutilise DB_DSN=inutilise DB_MIGRATION_DSN=inutilise \
       "$0" >/dev/null 2>&1; then
        echo "  ÉCHEC : le script a accepté un environnement non déclaré" >&2
        return 1
    fi
    echo "  refus confirmé"
    return 0
}

case "${1:-}" in
    --temoin) temoin; exit "$?" ;;
    "") ;;
    *) usage ;;
esac

# ── Le refus ─────────────────────────────────────────────────────────────────
case "${APP_ENV:-}" in
    development | test) ;;
    *)
        echo "refus : APP_ENV=${APP_ENV:-<absent>} — cette étape n'existe qu'en développement." >&2
        echo "        Hors development/test, les secrets viennent du mécanisme de" >&2
        echo "        l'environnement, jamais d'un fichier lu par le dépôt." >&2
        exit 1
        ;;
esac

for variable in DB_SUPERUSER_DSN DB_DSN DB_MIGRATION_DSN; do
    eval "valeur=\${$variable:-}"
    if [ -z "$valeur" ]; then
        echo "refus : $variable est vide — voir .env.example" >&2
        exit 1
    fi
done

# ── Extraction du couple rôle/mot de passe d'un DSN ──────────────────────────
#
# `postgres://ROLE:SECRET@hote:port/base?options` → « ROLE SECRET ».
#
# Le découpage se fait sur le PREMIER `@`, donc un mot de passe qui en contient
# un serait mal lu. Plutôt que de deviner, on refuse en nommant le remède :
# l'encodage pour-cent est la forme correcte dans une URL, et elle est acceptée
# telle quelle par PostgreSQL.
pose() {
    reste=${1#*://}
    identifiants=${reste%%@*}
    hote=${reste#*@}

    case "$hote" in
        *@*)
            echo "refus : le DSN contient plusieurs « @ » — encoder le mot de passe (%40)" >&2
            return 1
            ;;
    esac
    case "$identifiants" in
        *:*) ;;
        *)
            echo "refus : le DSN ne porte pas de mot de passe — voir .env.example" >&2
            return 1
            ;;
    esac

    role=${identifiants%%:*}
    secret=${identifiants#*:}

    # `:"role"` cite un IDENTIFIANT, `:'secret'` cite un LITTÉRAL : c'est psql
    # qui échappe, pas nous. Un mot de passe contenant une apostrophe ne compose
    # donc aucune commande imprévue.
    #
    # ⚠️ Par l'ENTRÉE STANDARD, jamais par `-c` : psql n'interpole pas ses
    # variables dans une commande passée en argument. Avec `-c`, la commande
    # partait littéralement et le serveur répondait :
    #
    #   ERROR: syntax error at or near ":"
    #
    # Mesuré. Le message ne nomme pas la cause, et l'écrire ici évite de la
    # rechercher deux fois.
    psql "$DB_SUPERUSER_DSN" -v ON_ERROR_STOP=1 -q \
        -v role="$role" -v secret="$secret" <<'SQL'
ALTER ROLE :"role" WITH PASSWORD :'secret';
SQL
    echo "  mot de passe posé pour $role"
}

echo "mots de passe locaux (APP_ENV=$APP_ENV) :"
pose "$DB_DSN"
pose "$DB_MIGRATION_DSN"
