#!/usr/bin/env sh
# Refuse une modification des zones à HAUTE INERTIE sans trace écrite.
#
# Remplace un fichier CODEOWNERS, qui coderait des personnes plutôt que des
# règles. Toucher au règlement, aux primitives ou aux migrations exige un ADR
# dans le même diff — ou un label d'échappatoire, tracé et justifié.
#
# ─────────────────────────────────────────────────────────────────────────────
# Pourquoi ce garde est un SCRIPT et non du YAML en ligne
# ─────────────────────────────────────────────────────────────────────────────
# Tous ses frères le sont déjà — `verifie-mention-outillage.sh`,
# `verifie-version-de-go.sh`, `verifie-fuite-de-secrets.sh` — et chacun porte un
# mode `--temoin` qui prouve qu'il sait échouer (ADR 013). Celui-ci était le
# seul resté en ligne, donc le seul qu'on ne pouvait ni lancer en local, ni
# éprouver.
#
# Ce n'est pas une préférence de style : c'est ce qui a laissé passer le défaut
# ci-dessous.
#
# ─────────────────────────────────────────────────────────────────────────────
# Le défaut que ce script corrige
# ─────────────────────────────────────────────────────────────────────────────
# La version en ligne lisait les labels dans `github.event.pull_request.labels`,
# c'est-à-dire dans la charge de l'événement — FIGÉE au déclenchement du
# workflow. Un label posé APRÈS l'échec y est invisible, et une relance rejoue la
# même charge.
#
# Or poser le label après avoir vu l'échec est la SEULE façon dont on s'en sert :
# le message d'erreur dit lui-même « poser le label ». L'échappatoire documentée
# ne pouvait donc pas fonctionner — un garde dont le remède ne prend pas effet
# est un garde qu'on finit par retirer.
#
# Ici, l'état des labels est relu à CHAQUE exécution, donc une relance suffit.
#
# ─────────────────────────────────────────────────────────────────────────────
# Usage
# ─────────────────────────────────────────────────────────────────────────────
#   verifie-inertie.sh <base-sha> <head-sha>   contrôle réel
#   verifie-inertie.sh --temoin                prouve que le garde sait refuser
#                                              ET qu'il sait être satisfait
#
# Variables lues :
#   PR_NUMBER          numéro de la PR, pour relire les labels (facultatif)
#   INERTIE_FICHIERS   remplace la liste des fichiers modifiés (témoin)
#   INERTIE_LABELS     remplace la liste des labels (témoin)
set -eu

MOTIF_INERTIE='^(rules/|arch-go\.yml|\.golangci\.yml|internal/pkg/|migrations/)'
MOTIF_ADR='^documentation/adr/[0-9]{3}-'
LABEL_ECHAPPATOIRE='inertia:justified'

# fichiers_modifies rend la liste des fichiers du diff.
fichiers_modifies() {
    if [ -n "${INERTIE_FICHIERS:-}" ]; then
        printf '%s\n' "$INERTIE_FICHIERS"
        return
    fi
    git diff --name-only "$1" "$2"
}

# labels_courants relit l'état des labels, MAINTENANT.
#
# Volontairement tolérant à l'échec : sans `gh`, sans jeton, ou hors d'une PR, la
# liste est vide et le garde retombe sur l'exigence d'ADR. Un garde qui
# planterait faute de pouvoir lire les labels serait plus fragile que le défaut
# qu'il surveille.
labels_courants() {
    if [ -n "${INERTIE_LABELS:-}" ]; then
        printf '%s\n' "$INERTIE_LABELS"
        return
    fi
    [ -n "${PR_NUMBER:-}" ] || return 0
    gh pr view "$PR_NUMBER" --json labels --jq '.labels[].name' 2>/dev/null || true
}

# echappatoire_posee cherche le label, en correspondance EXACTE.
#
# `grep -q 'inertia:justified'` accepterait un label `inertia:justified-plus-tard`
# ou n'importe quel libellé le contenant. Le motif est donc ancré aux deux bouts.
echappatoire_posee() {
    labels_courants | grep -qx "$LABEL_ECHAPPATOIRE"
}

# controle applique la règle et rend 0 ou 1.
controle() {
    changed=$(fichiers_modifies "${1:-}" "${2:-}")

    inertie=$(printf '%s\n' "$changed" | grep -E "$MOTIF_INERTIE" || true)
    if [ -z "$inertie" ]; then
        echo "Aucune zone à haute inertie touchée."
        return 0
    fi

    echo "Zones à haute inertie modifiées :"
    printf '%s\n' "$inertie" | sed 's/^/  · /'

    adr=$(printf '%s\n' "$changed" | grep -E "$MOTIF_ADR" || true)
    if [ -n "$adr" ]; then
        echo "ADR présent dans la PR :"
        printf '%s\n' "$adr" | sed 's/^/  · /'
        return 0
    fi

    if echappatoire_posee; then
        echo "Pas d'ADR, mais le label '$LABEL_ECHAPPATOIRE' est posé."
        echo "La justification doit figurer dans le corps de la PR."
        return 0
    fi

    echo "ERREUR: cette PR modifie le règlement, les primitives ou les migrations sans ADR." >&2
    echo "Ajouter documentation/adr/{NNN}-{slug}.md, ou poser le label" >&2
    echo "'$LABEL_ECHAPPATOIRE' en expliquant pourquoi dans le corps de la PR," >&2
    echo "puis relancer ce job — l'état des labels est relu à chaque exécution." >&2
    return 1
}

# temoin prouve les DEUX moitiés du garde.
#
# Un garde qui refuse tout passerait le premier cas et serait inutilisable ; un
# garde qui accepte tout passerait les autres et ne garderait rien. Les quatre
# cas ensemble sont la seule preuve qu'il discrimine.
#
# Le troisième — l'échappatoire — est celui que rien n'éprouvait, et c'est
# précisément celui qui était cassé.
temoin() {
    echec=0

    _cas "hors zone d'inertie, sans rien" "cmd/server/main.go" "" 0 || echec=1
    _cas "zone d'inertie, sans ADR ni label" "internal/pkg/exit/exit.go" "" 1 || echec=1
    _cas "zone d'inertie, avec le label" "internal/pkg/exit/exit.go" "$LABEL_ECHAPPATOIRE" 0 || echec=1
    _cas "zone d'inertie, label approchant" "internal/pkg/exit/exit.go" "inertia:justified-plus-tard" 1 || echec=1
    _cas "zone d'inertie, avec un ADR" \
        "rules/tests.md
documentation/adr/018-un-exemple.md" "" 0 || echec=1

    [ "$echec" -eq 0 ] && echo "témoin complet : le garde refuse ET sait être satisfait"
    return "$echec"
}

# _cas exécute un scénario et compare le code de retour à l'attendu.
_cas() {
    nom=$1
    fichiers=$2
    labels=$3
    attendu=$4

    obtenu=0
    INERTIE_FICHIERS="$fichiers" INERTIE_LABELS="$labels" controle >/dev/null 2>&1 || obtenu=$?

    if [ "$obtenu" -ne "$attendu" ]; then
        echo "ERREUR témoin: $nom — attendu $attendu, obtenu $obtenu" >&2
        return 1
    fi
    echo "  ok  $nom (code $obtenu)"
    return 0
}

if [ "${1:-}" = "--temoin" ]; then
    temoin
    exit $?
fi

if [ $# -lt 2 ]; then
    echo "usage: $0 <base-sha> <head-sha> | $0 --temoin" >&2
    exit 2
fi

controle "$1" "$2"
