#!/usr/bin/env sh
# Refuse un marqueur de dette dans les lignes AJOUTÉES d'une PR.
#   rules/interdictions.md § Dette · ADR 013
#
#   verifie-dette.sh <base> [head]   contrôle réel
#   verifie-dette.sh --temoin        prouve qu'il sait refuser ET être satisfait
#
# ─────────────────────────────────────────────────────────────────────────────
# Ce que la règle dit, et pourquoi elle est plus forte qu'elle n'en a l'air
# ─────────────────────────────────────────────────────────────────────────────
# Ce qui n'est pas fait s'annonce **hors périmètre dans la PR**, jamais en
# marqueur dans le code. La nuance n'est pas de rangement : un `TODO` est lu par
# la personne qui ouvre le fichier — c'est-à-dire par personne — alors qu'un
# hors-périmètre écrit dans une PR est lu par celle qui relit, au moment où elle
# peut encore dire non.
#
# ─────────────────────────────────────────────────────────────────────────────
# Pourquoi ce garde est un SCRIPT, et pourquoi si tard
# ─────────────────────────────────────────────────────────────────────────────
# Il était le DERNIER garde encore écrit en YAML dans `ci.yml` — audit #107,
# écart É-03. Tous ses frères vivent dans `tools/` et portent un `--temoin`
# depuis l'ADR 013. Lui n'en avait aucun, donc rien ne prouvait qu'il savait
# encore refuser, et on ne pouvait pas le lancer avant de pousser.
#
# Le précédent est trop précis pour être ignoré : le garde d'inertie était dans
# cet état, et son extraction (#111) a révélé que son échappatoire documentée
# **ne pouvait pas fonctionner** — un défaut présent depuis l'origine, invisible
# tant que le garde restait en ligne.
#
# ─────────────────────────────────────────────────────────────────────────────
# Ce que l'extraction a corrigé au passage — écart É-11
# ─────────────────────────────────────────────────────────────────────────────
# La version en ligne filtrait `'*.go' '*.sql' '*.yml' '*.yaml'`. Un marqueur
# posé dans un script de `tools/`, dans une règle de `rules/` ou dans un ADR
# passait donc sans bruit.
#
# Aucun n'existait — c'est vérifié — mais le garde ne le devait pas à sa
# couverture. Il balaie désormais TOUT le dépôt, avec une liste d'exclusions
# nommée plutôt qu'un filtre d'extensions muet.
set -eu

# MOTIF — assemblé, jamais écrit d'un seul tenant.
#
# Même raison que dans `verifie-mention-outillage.sh` : écrit en clair, il
# apparaîtrait dans le fichier qui le définit, et ce fichier se ferait signaler
# par le garde qu'il porte. La version en ligne de `ci.yml` avait exactement ce
# défaut, et c'est pour cela qu'elle devait s'exclure elle-même.
MOTIF="TO""DO|FIX""ME|X""XX|HA""CK"

# exclusions — un chemin, puis son motif, sur une ligne.
#
# Une exclusion répond à UNE question : ce fichier porte-t-il le motif parce
# qu'il DÉFINIT la règle, ou parce qu'il la VIOLE ? Seul le premier cas
# s'exclut. Dans le doute, il n'y a pas de doute.
exclusions() {
    cat <<'FIN'
tools/verifie-dette.sh porte le motif qu'il fait respecter — c'est sa définition même
tools/testdata/dette témoin d'échec (ADR 013) — faux exprès, ne jamais « corriger »
rules/interdictions.md énonce la règle, et une règle doit pouvoir nommer ce qu'elle interdit
FIN
}

# anti_pourriture refuse une exclusion devenue inutile.
#
# Une liste d'exceptions qui ne se relit pas devient une liste de passe-droits.
# Le jour où un fichier exclu cesse de porter le motif, la CI EXIGE qu'on retire
# son exclusion — sans quoi elle protégerait quelque chose qui n'existe plus.
#
# Ce contrôle a servi avant même d'être écrit : la première version de ce garde
# excluait `.github/workflows/ci.yml`, au motif que le job y nommait ce qu'il
# cherchait. L'extraction a déplacé le motif dans ce script, l'exclusion est
# devenue inutile le jour de sa naissance, et rien ne l'aurait dit.
anti_pourriture() {
    perimees=$(
        exclusions | while IFS=' ' read -r chemin _; do
            [ -n "$chemin" ] || continue
            if [ ! -e "$chemin" ]; then
                echo "$chemin (le chemin n'existe plus)"
            elif ! grep -rEq "$MOTIF" "$chemin" 2>/dev/null; then
                echo "$chemin (ne porte plus le motif)"
            fi
        done
    )
    if [ -n "$perimees" ]; then
        echo "  exclusion devenue inutile — la retirer de tools/verifie-dette.sh :" >&2
        printf '%s\n' "$perimees" | sed 's/^/    /' >&2
        return 1
    fi
    return 0
}

# lignes_ajoutees rend les lignes `+` du diff, hors en-têtes et chemins exclus.
#
# ⚠️ `base` et `head` sont capturés AVANT `set --`, qui écrase les paramètres
# positionnels. C'est le défaut exact qu'a eu `verifie-langue-du-code.sh` à sa
# première écriture : `git diff` recevait un `:(exclude)…` en guise de révision,
# échouait, et le `|| true` de fin transformait l'échec en « rien à signaler ».
lignes_ajoutees() {
    if [ -n "${DETTE_LIGNES:-}" ]; then
        printf '%s\n' "$DETTE_LIGNES"
        return 0
    fi

    base=$1
    head=$2

    set --
    while IFS=' ' read -r chemin _; do
        [ -n "$chemin" ] || continue
        set -- "$@" ":(exclude)$chemin"
    done <<FIN
$(exclusions)
FIN

    # Le diff est capturé AVANT d'être filtré : dans un tube, l'échec de `git`
    # serait masqué par le succès du dernier `grep`. Un garde qui rend 0 quand
    # son git échoue est un garde qui ne peut pas échouer.
    if ! diff_brut=$(git diff "$base" "$head" -- . "$@"); then
        echo "  git diff a échoué sur '$base'..'$head' — le garde n'a rien pu lire" >&2
        return 1
    fi

    printf '%s\n' "$diff_brut" | grep -E '^\+' | grep -Ev '^\+\+\+' || true
}

controle() {
    fail=0

    # Les exclusions sont affichées à CHAQUE exécution, avec leur motif : une
    # exception qu'on ne relit pas cesse d'être une exception.
    echo "  exclusions — chemins hors du champ du garde, avec leur motif :"
    exclusions | while IFS=' ' read -r chemin motif_texte; do
        [ -n "$chemin" ] || continue
        echo "    · $chemin"
        echo "        $motif_texte"
    done
    anti_pourriture || fail=1

    # Sans ce test, un échec de lecture du diff donnerait une liste vide, donc
    # un garde vert. C'est la forme exacte du faux vert que ce dépôt traque.
    if ! ajoutees=$(lignes_ajoutees "${1:-}" "${2:-}"); then
        return 1
    fi

    marqueurs=$(printf '%s\n' "$ajoutees" | grep -En "$MOTIF" || true)
    if [ -n "$marqueurs" ]; then
        echo "  marqueur de dette introduit :" >&2
        printf '%s\n' "$marqueurs" | sed 's/^/    /' >&2
        echo "" >&2
        echo "ERREUR: ce qui n'est pas fait s'annonce HORS PÉRIMÈTRE dans le corps de la" >&2
        echo "PR, pas en marqueur dans le code — rules/interdictions.md § Dette." >&2
        echo "Un marqueur est lu par qui ouvre le fichier, donc par personne ; un" >&2
        echo "hors-périmètre est lu par qui relit, au moment où il peut dire non." >&2
        return 1
    fi

    [ "$fail" -eq 0 ] && echo "  aucun marqueur de dette dans les lignes ajoutées."
    return "$fail"
}

# temoin prouve les DEUX moitiés du garde.
#
# Un garde qui refuse tout passerait les cas de refus et serait inutilisable ;
# un garde qui accepte tout passerait le dernier et ne garderait rien. Les cinq
# ensemble sont la seule preuve qu'il DISCRIMINE.
#
# La leçon vient de `verifie-mention-outillage.sh` : son volet sur les noms de
# fichiers a d'abord passé son cas de REFUS pour la mauvaise raison — la
# fonction de contrôle n'existait pas, elle rendait « commande introuvable »,
# donc un code non nul, donc « refus ». C'est le cas qui doit être ACCEPTÉ qui a
# révélé la panne.
temoin() {
    echec=0

    _cas "marqueur TO-DO" "+// $(printf 'TO')DO: revenir dessus" 1 || echec=1
    _cas "marqueur FIX-ME" "+// $(printf 'FIX')ME plus tard" 1 || echec=1
    _cas "marqueur HA-CK" "+	// $(printf 'HA')CK: contournement" 1 || echec=1
    _cas "ligne ordinaire" "+// NewUser builds a user." 0 || echec=1
    _cas_git "revision inexistante" 1 || echec=1

    [ "$echec" -eq 0 ] && echo "  témoin complet : le garde refuse ET sait être satisfait"
    return "$echec"
}

# _cas_git éprouve le chemin où le garde LIT le dépôt, que `_cas`
# court-circuite. Sans lui, le défaut « bad revision rendu en succès » ne serait
# attrapé par aucun témoin.
_cas_git() {
    nom=$1
    attendu=$2

    obtenu=0
    controle "revision-qui-n-existe-pas-0000" "HEAD" >/dev/null 2>&1 || obtenu=$?

    if [ "$obtenu" -ne "$attendu" ]; then
        echo "  ERREUR témoin: $nom — attendu $attendu, obtenu $obtenu" >&2
        return 1
    fi
    echo "  ok  $nom (code $obtenu)"
    return 0
}

_cas() {
    nom=$1
    contenu=$2
    attendu=$3

    obtenu=0
    DETTE_LIGNES="$contenu" controle >/dev/null 2>&1 || obtenu=$?

    if [ "$obtenu" -ne "$attendu" ]; then
        echo "  ERREUR témoin: $nom — attendu $attendu, obtenu $obtenu" >&2
        return 1
    fi
    echo "  ok  $nom (code $obtenu)"
    return 0
}

if [ "${1:-}" = "--temoin" ]; then
    temoin
    exit $?
fi

[ $# -ge 1 ] || { echo "usage: $0 <base> [head] | $0 --temoin" >&2; exit 2; }
controle "$1" "${2:-HEAD}"
