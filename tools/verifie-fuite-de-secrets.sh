#!/usr/bin/env sh
#
# Garde « aucun secret dans un artefact versionné » — rules/interdictions.md.
#
#   ./tools/verifie-fuite-de-secrets.sh            sens succès — le dépôt réel
#   ./tools/verifie-fuite-de-secrets.sh --temoin   sens échec  — ADR 013
#
# ─────────────────────────────────────────────────────────────────────────────
# UNE seule définition, appelée par `ci.yml` ET par `task ci:secrets`
# ─────────────────────────────────────────────────────────────────────────────
# Troisième garde ramené à une définition unique, après l'isolation des schémas
# (#40) et la mention d'outillage (#58). Le motif est toujours le même : deux
# copies d'un garde, c'est un garde et demi, et on ne sait jamais laquelle des
# deux moitiés a tourné.
#
# ─────────────────────────────────────────────────────────────────────────────
# Pourquoi il fallait le reprendre — #67
# ─────────────────────────────────────────────────────────────────────────────
# La CI appelait `gitleaks/gitleaks-action@v2`. Sur une `pull_request`, cette
# action prend un chemin qui interroge l'API GitHub. Le workflow applique le
# moindre privilège (`contents: read`), l'appel est refusé, le processus Node
# meurt — et AUCUN fichier n'est analysé :
#
#     event type: pull_request
#     RequestError [HttpError]: Resource not accessible by integration
#         at async Object.ScanPullRequest (…/gitleaks-action/v2/…)
#
# Sur `push`, l'autre chemin n'appelle pas l'API : le job était donc vert sur
# `main`. Et aucune PR n'avait jamais déclenché la CI (#64). Le seul chemin
# cassé était exactement le seul chemin jamais exercé.
#
# Neuvième garde du dépôt trouvé défectueux, et toujours la même forme : il ne
# signalait pas son inaction. Lancer le BINAIRE supprime la cause — pas d'appel
# à l'API, donc pas de privilège à élargir pour faire taire le garde.

set -eu

# Le dépôt entier, historique compris : un secret retiré au commit suivant reste
# dans l'historique, donc reste compromis. `--exit-code 1` est explicite parce
# que le défaut de gitleaks a déjà changé, et un scanner qui rend 0 sur une
# fuite est pire que pas de scanner.
scanne_depot() {
  gitleaks detect --source=. --redact --no-banner --exit-code 1
}

# `--no-git` : le bac à sable du témoin n'est pas un dépôt.
scanne_dossier() {
  gitleaks detect --source="$1" --no-git --redact --no-banner --exit-code 1
}

command -v gitleaks >/dev/null 2>&1 || {
  echo "  gitleaks est introuvable — l'installer AVANT d'appeler ce garde," >&2
  echo "  sinon son absence ressemblerait à un dépôt propre." >&2
  exit 127
}

# ─────────────────────────────────────────────────────────────────────────────
# Mode témoin — le scanner scanne-t-il encore ?
# ─────────────────────────────────────────────────────────────────────────────
# Le témoin est FABRIQUÉ à l'exécution, jamais versionné : un faux secret dans
# l'arbre ferait rougir le scan réel, et l'exception qu'il faudrait alors écrire
# serait une porte ouverte permanente.
#
# Il est assemblé par CONCATÉNATION, en deux morceaux : la chaîne complète
# n'existe donc nulle part dans ce fichier, qui serait sinon le premier à faire
# rougir le scan réel. Même procédé que le motif du garde de mention (#58).
#
# ⚠️ DEUX pièges rencontrés en écrivant ce témoin, tous deux dans le sens
#    « le garde a l'air mort alors qu'il fonctionne » — le pire sens, celui qui
#    conduit à « réparer » ce qui marche :
#
#    1. La clé d'EXEMPLE de la documentation d'AWS (`AKIA` + `IOSFODNN7EXAMPLE`)
#       n'est PAS détectée : gitleaks met délibérément en liste blanche tout ce
#       qui contient « EXAMPLE ».
#    2. Un corps TIRÉ AU HASARD tombe régulièrement sous le seuil d'entropie de
#       la règle `aws-access-token` dès qu'il répète des caractères — le témoin
#       devient alors intermittent. Un garde qui rougit au hasard finit
#       désactivé, ce qui est pire que pas de garde.
#
#    D'où un corps FIXE, à 16 caractères tous distincts : entropie mesurée
#    4,12 pour un seuil de 3,5, avec de la marge et sans aléa.
if [ "${1:-}" = "--temoin" ]; then
  bac=$(mktemp -d)
  # shellcheck disable=SC2064
  trap "rm -rf '$bac'" EXIT INT TERM

  # Format `aws-access-token` : AKIA suivi de 16 caractères [A-Z0-9]. Ces
  # 16 caractères-là ne sont l'identifiant de rien ni de personne.
  prefixe="AKIA"
  corps="QZ3XM7VB""NKLPRDT5"
  printf 'aws_access_key_id = %s%s\n' "$prefixe" "$corps" > "$bac/identifiants.txt"

  if scanne_dossier "$bac" >/dev/null 2>&1; then
    echo "  LE GARDE NE GARDE PLUS : le faux secret est passé inaperçu." >&2
    echo "  Tant que ceci est vrai, « aucune fuite » ne veut rien dire." >&2
    exit 1
  fi
  echo "  le garde sait échouer."
  exit 0
fi

echo "  scan du dépôt, historique complet :"
scanne_depot
echo "  aucune fuite de secret."
