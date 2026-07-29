#!/usr/bin/env sh
#
# Garde « aucune mention d'outillage d'assistance dans un artefact versionné ».
#   rules/interdictions.md § Artefacts versionnés · rules/workflow-git.md §3
#
#   ./tools/verifie-mention-outillage.sh <base> [head]   sens succès
#   ./tools/verifie-mention-outillage.sh --temoin        sens échec (ADR 013)
#
# Variables d'environnement facultatives, pour le contexte de PR :
#   PR_TITLE, PR_BODY — analysées si elles sont non vides.
#
# ─────────────────────────────────────────────────────────────────────────────
# UNE seule définition, appelée par `ci.yml` ET par `task ci:inertia`
# ─────────────────────────────────────────────────────────────────────────────
# Même motif que le garde d'isolation des schémas (#40) : le dépôt a déjà payé
# la divergence entre local et CI. Deux copies d'un garde, c'est un garde et
# demi — et on ne sait jamais laquelle des deux moitiés a tourné.
#
# ─────────────────────────────────────────────────────────────────────────────
# Pourquoi il fallait le reprendre — #58
# ─────────────────────────────────────────────────────────────────────────────
# Un garde de contenu a un angle mort de naissance : les fichiers qui ÉNONCENT
# la règle doivent nommer ce qu'elle interdit. Ils portent donc le motif, et se
# font signaler par le garde qu'ils définissent.
#
# Mesuré, pas supposé : rejoué sur la branche intégrée contre `origin/main`, il
# accusait `.githooks/commit-msg` — le crochet qui fait respecter l'interdiction.
# La fusion vers `main` aurait échoué sur ce job, et le seul remède à portée de
# main aurait été d'assouplir le motif. C'est-à-dire de désarmer la règle 🔴 du
# dépôt pour faire passer son propre garde.
#
# Remède : une liste d'exclusions NOMMÉE, MOTIVÉE et AFFICHÉE à chaque
# exécution — jamais un motif plus permissif. Trois choses empêchent que ce soit
# un contournement :
#
#   1. l'exclusion est par CHEMIN, jamais par motif : le motif reste intact ;
#   2. chaque exclusion porte son motif écrit, affiché à chaque exécution ;
#   3. un garde ANTI-POURRITURE refuse une exclusion devenue inutile — le jour
#      où un fichier exclu cesse de porter le motif, la CI EXIGE qu'on retire
#      son exclusion. Une liste d'exceptions qui ne se relit pas devient une
#      liste de passe-droits.
#
# ⚠️ Ce garde est livré avec le cas qui le fait ÉCHOUER (ADR 013) :
#    tools/testdata/mention-outillage/ — voir `--temoin`.

set -eu

# ─────────────────────────────────────────────────────────────────────────────
# Le motif — ASSEMBLÉ, jamais écrit d'un seul tenant
# ─────────────────────────────────────────────────────────────────────────────
# Écrit en clair, il apparaîtrait dans le fichier qui le définit, et toute PR
# touchant à ce garde se ferait signaler par le garde lui-même. La version de
# `ci.yml` portait exactement ce défaut.
#
# L'emoji est construit depuis ses OCTETS UTF-8 (F0 9F A4 96), comme dans
# `.githooks/commit-msg` et pour la même raison : écrit en clair, un outil
# Windows l'a déjà ré-encodé une fois, et le garde a cessé de reconnaître ce
# qu'il devait refuser — sans que rien ne le signale.
robot=$(printf '\360\237\244\226')
MOTIF="Co-""Authored-By:.*(claude|copilot|cursor|gpt|gemini|codex)"
MOTIF="$MOTIF|Generated ""with|Co-""authored-by:.*noreply@anthropic|${robot}"

# ─────────────────────────────────────────────────────────────────────────────
# Les exclusions — un chemin, puis son motif, sur une ligne
# ─────────────────────────────────────────────────────────────────────────────
# Toute ligne ajoutée ici doit répondre à UNE question : ce fichier porte-t-il
# le motif parce qu'il DÉFINIT la règle, ou parce qu'il la VIOLE ? Seul le
# premier cas s'exclut. Dans le doute, il n'y a pas de doute.
exclusions() {
  cat <<'FIN'
.githooks/commit-msg porte le motif qu'il fait respecter — c'est sa définition même
rules/workflow-git.md énonce la règle, et une règle doit pouvoir nommer ce qu'elle interdit
tools/testdata/mention-outillage témoin d'échec (ADR 013) — faux exprès, ne jamais « corriger »
FIN
}

# ─────────────────────────────────────────────────────────────────────────────
# 5. Les NOMS des fichiers versionnés
# ─────────────────────────────────────────────────────────────────────────────
# L'angle mort qui a duré le plus longtemps : le fichier d'amorçage du dépôt a
# porté le nom d'un outil d'assistance pendant toute la phase 0, à la racine,
# donc en première position pour quiconque ouvrait le dépôt.
#
# Aucun garde ne pouvait le voir. Celui-ci cherchait son motif dans le CONTENU,
# et il écarte explicitement les en-têtes `+++ b/…` du diff — commentaire
# d'origine : « éviter d'accuser un fichier pour son propre nom ». La ligne était
# juste pour le cas qu'elle visait ; elle a rendu celui-ci invisible.
#
# ⚠️ Une liste ÉNUMÉRÉE, pas un motif. Un motif sur les noms accuserait
# `internal/pkg/pagination/tests/cursor_round_trips_test.go`, où « cursor » est
# le mot anglais du domaine. Un garde qui crie au loup sur du code légitime finit
# désarmé — et il l'aurait été avant d'avoir servi une seule fois.
#
# La liste grandit quand un outil impose une nouvelle convention de fichier.
# C'est le mécanisme prévu, et il coûte une ligne.
artefacts_outillage() {
  cat <<'FIN'
claude.md
agents.md
.cursorrules
.windsurfrules
.aider.conf.yml
.github/copilot-instructions.md
FIN
}

# fichiers_versionnes rend la liste à contrôler. Surchargeable pour le témoin :
# sans cela, prouver que ce volet sait refuser exigerait de commettre un fichier
# interdit — donc de violer la règle pour démontrer qu'elle tient.
fichiers_versionnes() {
  if [ -n "${MENTION_FICHIERS:-}" ]; then
    printf '%s\n' "$MENTION_FICHIERS"
    return
  fi
  git ls-files
}

# controle_noms rend 0 si aucun nom versionné n'est un artefact d'outillage.
#
# La comparaison porte sur le nom de BASE autant que sur le chemin complet :
# déplacer un fichier interdit dans un sous-dossier ne doit pas le blanchir.
controle_noms() {
  noms=$(
    fichiers_versionnes | while read -r chemin; do
      minuscule=$(printf '%s' "$chemin" | tr '[:upper:]' '[:lower:]')
      artefacts_outillage | while read -r interdit; do
        [ -n "$interdit" ] || continue
        case "$minuscule" in
          "$interdit"|*/"$interdit") echo "$chemin" ;;
        esac
      done
    done
  )
  if [ -n "$noms" ]; then
    echo "  un fichier versionné porte le nom d'un artefact d'outillage d'assistance :" >&2
    echo "$noms" | sed 's/^/    /' >&2
    echo "  la substance vit dans documentation/AMORCAGE.md, au nom neutre." >&2
    return 1
  fi
  return 0
}


# ─────────────────────────────────────────────────────────────────────────────
# Mode témoin — le garde sait-il encore échouer ?
# ─────────────────────────────────────────────────────────────────────────────
# Rend 0 quand le garde détecte CORRECTEMENT le témoin, non nul sinon. Sans
# cette moitié, on ne distingue pas « aucune mention » de « le motif ne
# reconnaît plus rien » — la confusion exacte qui a laissé huit gardes
# défectueux passer inaperçus dans ce dépôt.
if [ "${1:-}" = "--temoin" ]; then
  temoin="tools/testdata/mention-outillage"
  if [ ! -d "$temoin" ]; then
    echo "  le témoin $temoin a disparu : le garde n'est plus vérifiable" >&2
    exit 1
  fi

  manques=0
  # Chaque forme du motif a son fichier : si une seule cesse d'être reconnue,
  # on veut savoir LAQUELLE, pas seulement que « le témoin passe ».
  # Un témoin est un `.txt` ; le LISEZ-MOI voisin explique, il ne témoigne pas.
  for f in "$temoin"/*.txt; do
    [ -f "$f" ] || continue
    if grep -Eiq "$MOTIF" "$f"; then
      echo "  détecté : $(basename "$f")"
    else
      echo "  NON DÉTECTÉ : $f — le motif ne reconnaît plus cette forme" >&2
      manques=1
    fi
  done

  # Le volet 5 — les NOMS de fichiers — a son propre témoin, et il ne peut pas
  # être un fichier : commettre un artefact interdit pour prouver qu'on le
  # refuse reviendrait à violer la règle pour la démontrer. La liste est donc
  # injectée.
  #
  # Les deux cas ensemble sont la seule preuve que ce volet DISCRIMINE : sans le
  # second, un garde qui refuserait tous les noms passerait le premier.
  if ! MENTION_FICHIERS="docs/CLAUDE.md" "$0" --temoin-noms >/dev/null 2>&1; then
    echo "  noms : un artefact d'outillage est correctement refusé"
  else
    echo "  NON DÉTECTÉ : un nom d'artefact d'outillage est passé" >&2
    manques=1
  fi
  if MENTION_FICHIERS="internal/pkg/pagination/tests/cursor_round_trips_test.go" \
       "$0" --temoin-noms >/dev/null 2>&1; then
    echo "  noms : un nom légitime portant « cursor » est accepté"
  else
    echo "  FAUX POSITIF : un nom légitime est refusé — le garde crie au loup" >&2
    manques=1
  fi

  if [ "$manques" -ne 0 ]; then
    echo "  LE GARDE NE GARDE PLUS." >&2
    exit 1
  fi
  echo "  le garde sait échouer."
  exit 0
fi

# --temoin-noms n'exécute QUE le volet 5, sur la liste injectée. Réservé au mode
# témoin ci-dessus ; il n'a pas de sens en usage direct.
if [ "${1:-}" = "--temoin-noms" ]; then
  controle_noms
  exit $?
fi

BASE="${1:-}"
HEAD="${2:-HEAD}"
[ -n "$BASE" ] || { echo "usage : $0 <base> [head] | --temoin" >&2; exit 2; }

fail=0

# ─────────────────────────────────────────────────────────────────────────────
# 1. Les exclusions sont-elles encore justifiées ? (garde anti-pourriture)
# ─────────────────────────────────────────────────────────────────────────────
echo "  exclusions — chemins hors du champ du garde, avec leur motif :"
exclusions | while IFS=' ' read -r chemin motif_texte; do
  [ -n "$chemin" ] || continue
  echo "    · $chemin"
  echo "        $motif_texte"
done

perimees=$(
  exclusions | while IFS=' ' read -r chemin _; do
    [ -n "$chemin" ] || continue
    if [ ! -e "$chemin" ]; then
      echo "$chemin (le chemin n'existe plus)"
    elif ! grep -rEiq "$MOTIF" "$chemin" 2>/dev/null; then
      echo "$chemin (ne porte plus le motif)"
    fi
  done
)
if [ -n "$perimees" ]; then
  echo "" >&2
  echo "  exclusion devenue inutile — la retirer de tools/verifie-mention-outillage.sh :" >&2
  echo "$perimees" | sed 's/^/    · /' >&2
  fail=1
fi

# Les chemins exclus, en pathspec git.
set --
while IFS=' ' read -r chemin _; do
  [ -n "$chemin" ] || continue
  set -- "$@" ":(exclude)$chemin"
done <<FIN
$(exclusions)
FIN

echo ""

# ─────────────────────────────────────────────────────────────────────────────
# 2. Les messages de commit — AUCUNE exclusion possible
# ─────────────────────────────────────────────────────────────────────────────
# Un message de commit n'a pas de chemin : il ne peut donc pas énoncer la règle,
# seulement la violer. Pas d'exception ici, et il ne faut pas en inventer.
if git log --format='%B' "$BASE..$HEAD" | grep -Eiq "$MOTIF"; then
  echo "  mention d'outillage d'assistance dans un message de commit" >&2
  fail=1
fi

# ─────────────────────────────────────────────────────────────────────────────
# 3. Le diff — lignes AJOUTÉES seulement, hors chemins exclus
# ─────────────────────────────────────────────────────────────────────────────
# `+++ b/…` est un en-tête de diff, pas une ligne de contenu : l'écarter évite
# d'accuser un fichier pour son propre nom.
touches=$(
  git diff "$BASE" "$HEAD" -- . "$@" \
    | grep -E '^\+' | grep -Ev '^\+\+\+' \
    | grep -Ei "$MOTIF" || true
)
if [ -n "$touches" ]; then
  echo "  le diff introduit une mention d'outillage d'assistance :" >&2
  echo "$touches" | sed 's/^/    /' >&2
  fail=1
fi

# ─────────────────────────────────────────────────────────────────────────────
# 4. Le titre et le corps de la PR, s'ils sont fournis
# ─────────────────────────────────────────────────────────────────────────────
if [ -n "${PR_TITLE:-}${PR_BODY:-}" ]; then
  if printf '%s\n%s' "${PR_TITLE:-}" "${PR_BODY:-}" | grep -Eiq "$MOTIF"; then
    echo "  le titre ou le corps de la PR porte une mention d'outillage d'assistance" >&2
    fail=1
  fi
fi

if ! controle_noms; then
  fail=1
fi

if [ "$fail" -eq 0 ]; then
  echo "  aucune mention d'outillage d'assistance."
fi
exit "$fail"
