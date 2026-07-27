#!/usr/bin/env sh
#
# Garde « la version de Go ne diverge pas » — #66.
#
#   ./tools/verifie-version-de-go.sh [go.mod] [Dockerfile] [Containerfile]
#
# Sans argument, les trois fichiers réels du dépôt.
#
# ─────────────────────────────────────────────────────────────────────────────
# Pourquoi ce garde existe
# ─────────────────────────────────────────────────────────────────────────────
# La version de Go est déclarée à TROIS endroits :
#
#   go.mod                          la version du LANGAGE — fait foi
#   Dockerfile                      l'image qui compile ce qui est LIVRÉ
#   deploy/toolbox/Containerfile    l'image où travaille l'équipe
#
# Rien ne les tenait ensemble. Une montée de version proposée sur un seul
# fichier — c'est exactement ce qu'a proposé la PR #63 — ferait développer
# l'équipe sous une version et livrer un binaire compilé sous une autre.
#
# Et la divergence est SILENCIEUSE par construction : `go build` accepte
# volontiers une chaîne d'outils plus récente que la directive `go` de `go.mod`.
# Rien ne casse, rien ne prévient, jusqu'au jour où un comportement diffère.
#
# Même classe de défaut que les trois seuils de couverture en trois endroits,
# dont un mort. Une règle non outillée n'existe pas.
#
# ─────────────────────────────────────────────────────────────────────────────
# Ce qui est comparé, et ce qui ne l'est pas
# ─────────────────────────────────────────────────────────────────────────────
# La comparaison porte sur `majeure.mineure` (`1.25`), pas sur le correctif.
# `go.mod` épingle `1.25.12` tandis que l'étiquette d'image `golang:1.25-alpine`
# suit le dernier correctif de la série : exiger l'égalité stricte rendrait ce
# garde rouge à chaque publication de Go, donc il finirait désactivé.
#
# ⚠️ Ce garde est livré avec le cas qui le fait ÉCHOUER (ADR 013) :
#    tools/testdata/version-de-go-divergente/ — voir `task ci:go-version`.

set -eu

GOMOD="${1:-go.mod}"
DOCKERFILE="${2:-Dockerfile}"
CONTAINERFILE="${3:-deploy/toolbox/Containerfile}"

# `go 1.25.12` → `1.25`, `golang:1.25-alpine` → `1.25`.
majeure_mineure() {
  grep -oE "$2" "$1" | head -n1 | grep -oE '[0-9]+\.[0-9]+' | head -n1
}

version_gomod=$(majeure_mineure "$GOMOD" '^go [0-9]+\.[0-9]+(\.[0-9]+)?')
version_image=$(majeure_mineure "$DOCKERFILE" 'golang:[0-9]+\.[0-9]+')
version_toolbox=$(majeure_mineure "$CONTAINERFILE" 'golang:[0-9]+\.[0-9]+')

fail=0
for couple in "go.mod:$version_gomod" "image de livraison:$version_image" \
              "toolbox:$version_toolbox"; do
  quoi=${couple%%:*}; valeur=${couple#*:}
  if [ -z "$valeur" ]; then
    echo "  version de Go illisible dans « $quoi »" >&2
    fail=1
  else
    printf '    %-22s %s\n' "$quoi" "$valeur"
  fi
done
[ "$fail" -eq 0 ] || exit 1

if [ "$version_gomod" = "$version_image" ] && [ "$version_gomod" = "$version_toolbox" ]; then
  echo "  version de Go cohérente : $version_gomod"
  exit 0
fi

echo "" >&2
echo '  LES VERSIONS DE GO DIVERGENT.' >&2
echo '  go.mod fait foi. Monter de version se fait aux TROIS endroits a la' >&2
echo '  fois, dans le meme diff -- sinon on developpe sous une version et on' >&2
echo '  livre sous une autre, sans que rien ne le dise.' >&2
exit 1
