#!/usr/bin/env sh
# Refuse une opération huma qui ne désarme pas la borne de corps de huma.
#   ADR 013 · issue #141
#
#   verifie-borne-de-corps.sh        contrôle réel
#   verifie-borne-de-corps.sh --temoin
#
# ─────────────────────────────────────────────────────────────────────────────
# Ce que ce garde empêche, et pourquoi rien d'autre ne le voyait
# ─────────────────────────────────────────────────────────────────────────────
# huma porte sa borne de corps sur CHAQUE `huma.Operation`, pas sur sa Config,
# et `huma.Register` la fixe silencieusement à 1 MiB quand l'entrée a un `Body`.
# Toutes les routes métier passent par huma : cette valeur, écrite nulle part,
# était donc la borne qui répondait réellement — pendant que
# `http.max_body_bytes` était lue, validée au démarrage, et sans le moindre
# effet.
#
# Le défaut a vécu des mois parce qu'il était EXACTEMENT entre deux choses
# testées : `middleware.MaxBody` avait ses tests et ils passaient, sur une route
# chi nue ; les routes métier avaient les leurs, et aucune ne portait sur la
# taille. Personne ne teste l'espace entre deux tests.
#
# Il n'a été trouvé qu'en MESURANT — la preuve de la persona P2 a réglé
# `max_body_bytes` à 50 MiB, posté 5 MiB, et reçu `413 … limit=1048576 bytes`.
#
# ─────────────────────────────────────────────────────────────────────────────
# Pourquoi un garde plutôt qu'une convention
# ─────────────────────────────────────────────────────────────────────────────
# Le remède tient en un champ par opération. C'est précisément ce qui le rend
# fragile : une route neuve qui l'oublie ne casse rien, ne loge rien, et retombe
# à 1 MiB sans que personne ne l'apprenne avant un import en masse en
# production.
#
# Le gabarit du générateur le porte aussi : sans cela, CHAQUE projet engendré
# naîtrait avec le défaut.
set -eu

MOTIF_REGISTER='huma\.Register('
MOTIF_BORNE='MaxBodyBytes:'

# fichiers rend les sources susceptibles d'enregistrer une opération.
#
# Les gabarits du générateur sont inclus : un défaut qui y vit se recopie dans
# tout projet engendré, ce qui est la forme la plus chère.
fichiers() {
    if [ -n "${BORNE_FICHIERS:-}" ]; then
        printf '%s\n' "$BORNE_FICHIERS"
        return 0
    fi
    git ls-files '*.go' '*.go.tmpl'
}

# controle compare, fichier par fichier, le nombre d'enregistrements au nombre
# de bornes désarmées.
#
# Comparer des COMPTES plutôt que d'analyser la syntaxe : un `huma.Operation`
# s'étend sur quinze lignes, et un analyseur de structures littérales en shell
# serait plus fragile que ce qu'il garde. Le compte suffit — il ne dit pas
# QUELLE opération manque, mais il ne peut pas passer à côté d'une qui manque.
controle() {
    manquants=$(
        fichiers | while read -r fichier; do
            [ -f "$fichier" ] || continue
            n_reg=$(grep -c "$MOTIF_REGISTER" "$fichier" 2>/dev/null || true)
            [ "${n_reg:-0}" -gt 0 ] || continue
            n_borne=$(grep -c "$MOTIF_BORNE" "$fichier" 2>/dev/null || true)
            if [ "${n_borne:-0}" -lt "$n_reg" ]; then
                echo "$fichier ($n_reg enregistrement(s), ${n_borne:-0} borne(s))"
            fi
        done
    )

    if [ -n "$manquants" ]; then
        echo "  opération huma sans borne de corps désarmée :" >&2
        printf '%s\n' "$manquants" | sed 's/^/    /' >&2
        echo "" >&2
        echo "ERREUR: huma fixe la borne à 1 MiB sur toute opération qui ne la" >&2
        echo "porte pas, et c'est ELLE qui répond — pas http.max_body_bytes." >&2
        echo "Ajouter dans le littéral huma.Operation :" >&2
        echo "    MaxBodyBytes: middleware.NoBodyLimit," >&2
        echo "Voir internal/pkg/middleware/max_body.go et l'issue #141." >&2
        return 1
    fi

    total=$(fichiers | while read -r f; do
        [ -f "$f" ] && grep -c "$MOTIF_REGISTER" "$f" 2>/dev/null || true
    done | awk '{s+=$1} END {print s+0}')
    echo "  $total opération(s) huma, toutes avec leur borne désarmée."
    return 0
}

# temoin prouve les DEUX moitiés.
#
# Le cas ACCEPTÉ compte autant que celui qui refuse : un garde qui refuse tout
# serait retiré à la première PR. Et c'est le cas passant qui révèle les pannes
# — `verifie-mention-outillage.sh` a d'abord passé son cas de refus pour la
# mauvaise raison, une fonction inexistante rendant « commande introuvable ».
temoin() {
    echec=0
    fixtures=$(mktemp -d)

    printf 'huma.Register(api, huma.Operation{\n\tOperationID: "x",\n})\n' \
        >"$fixtures/sans.go"
    printf 'huma.Register(api, huma.Operation{\n\tOperationID: "x",\n\tMaxBodyBytes: middleware.NoBodyLimit,\n})\n' \
        >"$fixtures/avec.go"
    printf 'func Ordinary() {}\n' >"$fixtures/aucun.go"
    printf 'huma.Register(api, huma.Operation{\n\tMaxBodyBytes: middleware.NoBodyLimit,\n})\nhuma.Register(api, huma.Operation{\n\tOperationID: "y",\n})\n' \
        >"$fixtures/une_sur_deux.go"

    _cas "opération sans borne" "$fixtures/sans.go" 1 || echec=1
    _cas "opération avec borne" "$fixtures/avec.go" 0 || echec=1
    _cas "fichier sans opération" "$fixtures/aucun.go" 0 || echec=1
    # Le cas qui compte vraiment : un fichier qui en porte plusieurs et en
    # oublie UNE. C'est la forme qu'aura le vrai défaut — on ajoute une route à
    # un fichier qui en a déjà, et on copie tout sauf ce champ.
    _cas "une opération sur deux" "$fixtures/une_sur_deux.go" 1 || echec=1

    rm -rf "$fixtures"
    [ "$echec" -eq 0 ] && echo "  témoin complet : le garde refuse ET sait être satisfait"
    return "$echec"
}

_cas() {
    nom=$1
    fichier=$2
    attendu=$3

    obtenu=0
    BORNE_FICHIERS="$fichier" controle >/dev/null 2>&1 || obtenu=$?

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

controle
