#!/usr/bin/env sh
#
# Garde « aucun fichier suivi ne commence par un BOM UTF-8 » — #66.
#
#   ./tools/verifie-encodage.sh [dossier]
#
# ─────────────────────────────────────────────────────────────────────────────
# Pourquoi
# ─────────────────────────────────────────────────────────────────────────────
# `rules/toolchain.md` et le tableau des pièges d'outillage disent déjà
# « toujours écrire en UTF-8 SANS BOM ». La règle n'était outillée nulle part —
# et une règle non outillée n'existe pas. Résultat : `Dockerfile` portait un BOM
# (`EF BB BF`), seul fichier suivi dans ce cas, vestige de l'époque PowerShell.
#
# Ce n'est pas cosmétique. Un BOM place trois octets AVANT la première ligne :
#
#   · `# syntax=docker/dockerfile:1.7` doit être la PREMIÈRE ligne pour que
#     BuildKit l'honore. Précédée d'un BOM, elle ne l'est plus tout à fait ;
#   · un `#!/usr/bin/env sh` précédé d'un BOM n'est plus un shebang du tout ;
#   · le même dépôt a déjà payé 408 séquences d'accents doublement encodées
#     dans 8 fichiers, produites par la même chaîne d'outils.
#
# Le point commun de ces trois effets : ils ne produisent aucun message. Le
# fichier a l'air normal, l'outil fait autre chose que ce qu'on lit.
#
# ⚠️ Ce garde est livré avec le cas qui le fait ÉCHOUER (ADR 013) : voir
#    `--temoin`, dont le fichier fautif est FABRIQUÉ — un fichier à BOM
#    versionné serait précisément ce que ce garde interdit.

set -eu

if [ "${1:-}" = "--temoin" ]; then
  bac=$(mktemp -d)
  # shellcheck disable=SC2064
  trap "rm -rf '$bac'" EXIT INT TERM

  manques=0

  # Sens 1 — le BOM.
  printf '\357\273\277# un fichier qui ment sur sa premiere ligne\n' > "$bac/bom.txt"
  if head -c3 "$bac/bom.txt" | od -An -tx1 | tr -d ' \n' | grep -q '^efbbbf$'; then
    echo "  détecté : un BOM en tête de fichier"
  else
    echo "  NON DÉTECTÉ : le BOM — le garde ne reconnaît plus cette forme" >&2
    manques=1
  fi

  # Sens 2 — l'accent doublement encodé. Écrit en octets, comme le motif.
  printf 'un accent qui a fait l\047aller-retour : \303\203\n' > "$bac/mojibake.txt"
  printf 'un tiret cadratin parfaitement valide \342\200\224 a ne PAS accuser\n' \
    > "$bac/sain.txt"
  motif_temoin=$(printf '\303\203|\303\242\342\202\254|\303\242\342\200\235')
  if grep -qE "$motif_temoin" "$bac/mojibake.txt" \
     && ! grep -qE "$motif_temoin" "$bac/sain.txt"; then
    echo "  détecté : un accent doublement encodé, et le tiret sain épargné"
  else
    echo "  NON DÉTECTÉ : l'accent doublement encodé, ou faux positif sur un" >&2
    echo "  tiret cadratin valide — les deux rendent ce garde inutilisable" >&2
    manques=1
  fi

  [ "$manques" -eq 0 ] || { echo "  LE GARDE NE GARDE PLUS." >&2; exit 1; }
  echo "  le garde sait échouer."
  exit 0
fi

RACINE="${1:-.}"

# `git ls-files` plutôt qu'un `find` : seuls les fichiers SUIVIS engagent le
# dépôt, et ça écarte d'office `.git/`, `bin/` et les caches.
avec_bom=$(
  git -C "$RACINE" ls-files -z \
    | xargs -0 -r -I{} sh -c '
        head -c3 "{}" 2>/dev/null | od -An -tx1 | tr -d " \n" | grep -q "^efbbbf$" && echo "{}"
      ' || true
)

fail=0
if [ -n "$avec_bom" ]; then
  echo "  fichier(s) suivi(s) commençant par un BOM UTF-8 :" >&2
  echo "$avec_bom" | sed 's/^/    · /' >&2
  echo "  Réécrire en UTF-8 sans BOM (rules/toolchain.md)." >&2
  fail=1
fi

# ── Accents doublement encodés ───────────────────────────────────────────────
# Un « a » accent grave écrit en UTF-8, relu comme du latin-1, puis réécrit en
# UTF-8, donne deux caractères là où il y en avait un. Ces séquences
# n'apparaissent jamais dans du texte français légitime : les trouver, c'est
# trouver la trace d'un aller-retour d'encodage.
#
# Le dépôt en a déjà réparé 408 dans 8 fichiers. Aucun outil ne les cherchait,
# donc la 409ᵉ serait passée comme les précédentes — elles ne cassent rien,
# elles rendent seulement le texte faux.
#
# Le motif est construit depuis ses OCTETS, jamais écrit en clair : écrit tel
# quel, ce fichier serait le premier que son propre garde accuserait. Même
# procédé que le garde de mention d'outillage (#58) et pour la même raison.
#
#   303 203              « A » tilde majuscule, seul — jamais français
#   303 242 342 202 254  « a » circonflexe suivi du signe euro — la signature
#                        du tiret cadratin passé par un aller-retour latin-1
#   303 242 342 200 235  « a » circonflexe suivi d'un guillemet fermant — celle
#                        du filet de tableau
#
# ⚠️ Aucun de ces caractères n'est ÉCRIT dans ce fichier, seulement décrit et
#    codé en octal. La version précédente de ce commentaire les portait en
#    clair, et le garde s'accusait lui-même — trois lignes plus bas que la
#    phrase qui annonçait précisément ce risque. Il paraissait vert : le
#    fichier n'était pas encore SUIVI, et le garde n'inspecte que le suivi.
#
# ⚠️ DEUX essais faux avant celui-ci, dans les deux sens opposés :
#
#    1. chercher 342 200 et 342 224 : ce sont les deux PREMIERS octets de « — »
#       et « ─ » parfaitement valides. Le garde accusait 20 fichiers sains. Un
#       garde qui crie toujours ne garde rien — c'est exactement ce qui avait
#       vidé de son sens le garde d'isolation des schémas (#40) ;
#    2. chercher « â » suivi du seul octet 342 : la séquence UTF-8 est alors
#       INCOMPLÈTE. Le grep de la toolbox (busybox) refuse le motif entier avec
#       « Invalid regexp », donc le garde ne cherche plus RIEN — et il l'a fait
#       en local sans broncher, parce que le grep de l'hôte est plus tolérant.
#       Un garde doit être éprouvé là où la CI le lance, pas là où on l'écrit.
#
#    D'où des séquences COMPLÈTES. Et `â` seul reste légitime en français
#    (âge, château) : c'est ce qui le suit qui trahit le double encodage.
motif=$(printf '\303\203|\303\242\342\202\254|\303\242\342\200\235')

mojibake=$(git -C "$RACINE" grep -lE "$motif" -- . || true)
if [ -n "$mojibake" ]; then
  echo "  accent(s) doublement encodé(s) — trace d'un aller-retour d'encodage :" >&2
  git -C "$RACINE" grep -nE "$motif" -- . | sed 's/^/    /' >&2
  echo "  Réécrire les lignes fautives en UTF-8." >&2
  fail=1
fi

[ "$fail" -eq 0 ] && echo "  encodage sain : aucun BOM, aucun accent doublement encodé."
exit "$fail"
