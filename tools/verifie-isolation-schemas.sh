#!/usr/bin/env sh
#
# Garde d'isolation des schémas SQL — ADR 011.
#
#   ./tools/verifie-isolation-schemas.sh [dossier-migrations] [dossier-modules]
#
# `arch-go` empêche un module d'en IMPORTER un autre. Rien n'empêche son SQL
# d'aller lire les tables d'un autre : il n'y a aucun import à détecter. Ce
# script ferme cette porte-là.
#
# ─────────────────────────────────────────────────────────────────────────────
# UNE seule définition, appelée par `ci.yml` ET par `task ci:schemas`
# ─────────────────────────────────────────────────────────────────────────────
# Le dépôt a déjà payé la divergence entre local et CI — trois seuils de
# couverture en trois endroits, dont un mort. Ce garde ne la rejouera pas.
#
# ─────────────────────────────────────────────────────────────────────────────
# Pourquoi la version précédente ne gardait rien
# ─────────────────────────────────────────────────────────────────────────────
# Son motif s'appliquait au fichier ENTIER, commentaires et chaînes compris. Il
# accusait donc de la PROSE d'être un schéma étranger — `verify.sql`,
# `provision.sql`, `securite.md`, `user.registered` — et jusqu'aux séquences
# `\n` et `\r`. Rejoué sur le contenu de `main` : NEUF faux positifs, aucun
# vrai. Un garde qui crie toujours ne garde rien : on finit par le désactiver.
#
# Remède : ne comparer que le SQL EXÉCUTABLE. Les commentaires `--` et les
# chaînes littérales sont retirés avant l'analyse.
#
# Second défaut, plus discret : la boucle traitait le nom du dossier comme un
# MODULE, alors que `migrations/{moteur}/` porte un MOTEUR (ADR 012). Avec
# `migrations/postgres/`, la comparaison portait sur « postgres » et n'avait
# donc pas le sens qu'elle paraissait avoir. Les migrations du socle ne peuvent
# toucher que le schéma partagé `platform` : c'est ce qui est vérifié.
#
# ⚠️ Ce garde est livré avec le cas qui le fait ÉCHOUER (ADR 013) :
#    tools/testdata/isolation-violation/ — voir `task ci:schemas`.

set -eu

MIGRATIONS="${1:-migrations}"
MODULES="${2:-internal/modules}"

# Schémas qu'une migration du socle a le droit de nommer.
PARTAGES='platform|pg_catalog|information_schema|public|pg_temp'

fail=0

# ── 1. Les migrations ne nomment aucun schéma étranger ───────────────────────
if [ -d "$MIGRATIONS" ]; then
  for dir in "$MIGRATIONS"/*/; do
    [ -d "$dir" ] || continue
    moteur=$(basename "$dir")

    # Un fichier à la fois : sed dépouille, puis on cherche `schema.objet`.
    #   1. retire tout ce qui suit `--` (commentaire de ligne)
    #   2. vide le contenu des chaînes '…' — un libellé n'est pas du SQL
    etrangers=$(
      find "$dir" -name '*.sql' -type f -exec sh -c '
        for f do sed -e "s/--.*//" -e "s/'\''[^'\'']*'\''/'\'''\''/g" "$f"; done
      ' sh {} + 2>/dev/null \
        | grep -oE '\b[a-z][a-z0-9_]*\.[a-z][a-z0-9_]*' \
        | cut -d. -f1 | sort -u \
        | grep -Ev "^($PARTAGES)\$" || true
    )

    if [ -n "$etrangers" ]; then
      echo "migrations/$moteur nomme un schéma étranger :" >&2
      echo "$etrangers" | sed 's/^/    /' >&2
      fail=1
    fi
  done
fi

# ── 2. Aucun SQL applicatif ne qualifie le schéma d'un autre module ──────────
#
# Le `search_path` du rôle du module s'en charge : qualifier explicitement le
# schéma d'un autre module est le seul moyen de traverser la frontière.
if [ -d "$MODULES" ]; then
  modules=$(find "$MODULES" -maxdepth 1 -mindepth 1 -type d -exec basename {} \; | sort)
  for module in $modules; do
    for autre in $modules; do
      [ "$autre" = "$module" ] && continue
      trouve=$(grep -rInE "\b${autre}\.[a-z_]+" "$MODULES/$module" --include='*.go' 2>/dev/null || true)
      if [ -n "$trouve" ]; then
        echo "$module référence le schéma de $autre dans du SQL :" >&2
        echo "$trouve" | sed 's/^/    /' >&2
        fail=1
      fi
    done
  done
fi

if [ "$fail" -eq 0 ]; then
  echo "  isolation des schémas respectée."
fi
exit "$fail"
