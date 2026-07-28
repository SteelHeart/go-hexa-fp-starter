#!/usr/bin/env sh
#
# Garde « aucun déploiement sans CI verte sur le commit exact » — #75.
#
#   ./tools/attend-la-ci.sh <sha>
#   ./tools/attend-la-ci.sh --temoin
#
# ─────────────────────────────────────────────────────────────────────────────
# Ce que ce garde a fait pendant des semaines
# ─────────────────────────────────────────────────────────────────────────────
# Il échouait en 0,4 seconde, à chaque poussée sur `main`, en affichant :
#
#   ::error::CI en échec () — déploiement refusé
#
# Message faux. La CI était verte. Ce qui échouait, c'était l'INTERROGATION :
#
#   HTTP 403: Resource not accessible by integration
#     https://api.github.com/repos/…/actions/workflows
#
# Le job déclarait `permissions: contents: read`, et `gh run list` a besoin de
# `actions: read`. Exactement la faute de #67, où l'action gitleaks mourait sur
# un appel d'API que son jeton n'autorisait pas.
#
# Le défaut de FOND n'est pas la permission manquante, c'est que le garde
# répondait la même chose dans deux situations opposées :
#
#   « la CI est rouge »            → refuser est CORRECT
#   « je ne peux pas savoir »      → refuser est correct AUSSI, mais il faut le DIRE
#
# Confondues, la seconde se lit comme la première, on cherche la panne dans la
# CI, et le rouge permanent finit par s'ignorer tout seul.
#
# ─────────────────────────────────────────────────────────────────────────────
# Deny par défaut, mais motivé
# ─────────────────────────────────────────────────────────────────────────────
# Tout ce qui n'est pas un `success` constaté refuse le déploiement — y compris
# l'impossibilité d'interroger. Ce qui change, c'est que le motif est nommé.
#
# ⚠️ Ce garde est livré avec le cas qui le fait échouer (ADR 013) : `--temoin`
#    passe à `interprete` chacune des issues possibles et vérifie qu'aucune ne
#    laisse passer un déploiement.

set -eu

ESSAIS=${ESSAIS:-90}
ATTENTE=${ATTENTE:-20}

# ── interprète UNE conclusion ────────────────────────────────────────────────
#
# 0 → déployer · 1 → refuser · 2 → réessayer.
#
# Isolée du réseau exprès : c'est ce qui la rend exerçable par le témoin. La
# faute de #75 était ici, pas dans la boucle.
interprete() {
    case "$1" in
        success)
            echo "CI verte."
            return 0
            ;;
        pending)
            return 2
            ;;
        injoignable)
            echo "::error::impossible d'interroger la CI — déploiement refusé." >&2
            echo "  Ce n'est PAS « la CI est rouge ». Vérifier que le job déclare" >&2
            echo "  'permissions: actions: read' : gh run list en a besoin." >&2
            return 1
            ;;
        "")
            echo "::error::aucune exécution de CI pour ce commit — déploiement refusé" >&2
            return 1
            ;;
        *)
            echo "::error::CI en échec ($1) — déploiement refusé" >&2
            return 1
            ;;
    esac
}

# ── Témoin ───────────────────────────────────────────────────────────────────
temoin() {
    echec=0

    verifie() {
        attendu=$2
        obtenu=0
        interprete "$1" >/dev/null 2>&1 || obtenu=$?
        if [ "$obtenu" != "$attendu" ]; then
            echo "  ÉCHEC : « $1 » rend $obtenu, attendu $attendu" >&2
            echec=1
        else
            echo "  « ${1:-<vide>} » → $obtenu"
        fi
    }

    echo "témoin : seule une CI verte constatée autorise un déploiement"
    verifie success 0
    verifie pending 2
    verifie failure 1
    verifie cancelled 1
    verifie timed_out 1
    verifie startup_failure 1
    verifie injoignable 1
    verifie "" 1

    [ "$echec" -eq 0 ] || return 1
    echo "  le garde sait refuser, et distingue « rouge » de « injoignable »."
    return 0
}

case "${1:-}" in
    --temoin) temoin; exit "$?" ;;
    "") echo "usage: $0 <sha> | --temoin" >&2; exit 2 ;;
esac

SHA=$1

i=1
while [ "$i" -le "$ESSAIS" ]; do
    # `|| statut=injoignable` : sans cela, `set -e` tue le script sur l'échec de
    # la substitution, et la seule trace est un code de retour nu.
    statut=$(gh run list --commit "$SHA" --workflow CI --json status,conclusion \
        --jq '[.[] | select(.status=="completed")] | first | .conclusion // "pending"' \
        2>/dev/null) || statut=injoignable

    verdict=0
    interprete "$statut" || verdict=$?
    case "$verdict" in
        0) exit 0 ;;
        1) exit 1 ;;
    esac

    echo "CI en cours ($i/$ESSAIS)…"
    sleep "$ATTENTE"
    i=$((i + 1))
done

echo "::error::délai dépassé en attendant la CI — déploiement refusé" >&2
exit 1
