#!/usr/bin/env sh
# Refuse du français dans le CODE — ADR 018.
#
#   verifie-langue-du-code.sh <base> [head]   contrôle réel, lignes AJOUTÉES
#   verifie-langue-du-code.sh --temoin        prouve qu'il sait refuser ET être satisfait
#
# ─────────────────────────────────────────────────────────────────────────────
# Ce que ce garde NE fait PAS, et pourquoi c'est écrit ici
# ─────────────────────────────────────────────────────────────────────────────
# Il ne prouve pas qu'un texte est anglais. Aucun garde bon marché ne le peut :
# il faudrait un dictionnaire, et un dictionnaire refuserait la moitié des
# identifiants d'un domaine métier.
#
# Il attrape le français ÉCRIT COMME LE FRANÇAIS L'EST — accentué, ou puisant
# dans un vocabulaire observé dans ce dépôt. C'est une passoire à grosses
# mailles, et c'est assumé : elle attrape la faute de réflexe, celle qu'on
# commet en écrivant vite, qui est la seule que la revue laisse passer.
#
# La limite est écrite plutôt que laissée à découvrir. La liste de mots se
# complète à chaque faute trouvée en revue — c'est le mécanisme prévu, pas un
# aveu.
#
# ─────────────────────────────────────────────────────────────────────────────
# Pourquoi seulement les lignes AJOUTÉES
# ─────────────────────────────────────────────────────────────────────────────
# La traduction se livre en tranches (ADR 018). Un garde qui exigerait tout le
# dépôt d'un coup serait rouge jusqu'à la dernière tranche, donc ignoré, donc
# désarmé. En ne regardant que ce qu'une PR ajoute, il tient dès la première :
# il empêche la dette de croître pendant qu'on la résorbe.
set -eu

# Les zones de CODE, et les fichiers **Go** seulement.
#
# ⚠️ Le `.go` n'est pas un détail de commodité. L'ADR 018 énumère ce qui doit
# être en anglais — identifiants Go, godoc, messages d'erreur internes, contenu
# des tests — et ne mentionne **jamais** les scripts shell. Ce garde a d'abord
# balayé tout `tools/`, et il a immédiatement accusé ses propres frères : les
# gardes sont commentés en français, comme le règlement qu'ils font respecter.
#
# La ligne de partage de l'ADR est « ce qui est publié AVEC le code » : le godoc
# le sera, un commentaire de `verifie-inertie.sh` ne le sera jamais. Ces scripts
# s'adressent à l'équipe, exactement comme `rules/`.
#
# `tools/covergate`, écrit en Go, reste donc dans le champ — c'est du code
# produit par le dépôt, pas de la prose d'outillage.
ZONES='internal cmd tools'
EXTENSIONS='*.go'

# Marques diacritiques — écrites par leurs octets UTF-8, jamais en clair.
#
# Même raison que l'emoji de `verifie-mention-outillage.sh` : un outil Windows a
# déjà ré-encodé un motif de ce dépôt, et le garde a cessé de reconnaître ce
# qu'il devait refuser SANS que rien ne le signale. Un motif illisible qui
# fonctionne vaut mieux qu'un motif lisible qui a cessé de chercher.
ACCENTS=$(printf '\303\251\303\250\303\252\303\253\303\240\303\242\303\244\303\247\303\264\303\266\303\271\303\273\303\274\303\256\303\257\303\211\303\210\303\200\303\207')
MOTIF_ACCENT="[$ACCENTS]"

# Vocabulaire français observé DANS CE DÉPÔT, sans accent — donc invisible au
# contrôle ci-dessus. La recherche se fait sur des mots ENTIERS après passage
# par `segmente`, ce qui évite d'accuser `decomposer` pour le `composer` qu'il
# contient.
#
# Un seul mot par entrée, sans accent et sans souligné : `segmente` réduit tout
# identifiant à des mots séparés par des espaces, donc un motif composé ne
# correspondrait jamais.
MOTS='abonner|abonnement|amorcage|amorcer|boucler|boucle|composer'
MOTS="$MOTS|demarrer|depiler|depileur|envoyer|peupler|poser|lire|ecrire|refuser"
MOTS="$MOTS|verifier|verifie|creer|chercher|remplir|vider|attendre|arreter"
MOTS="$MOTS|fichier|dossier|chaine|jeton|sujet|utilisateur|passe"
MOTS="$MOTS|inscrire|inscription|nombre|erreur|essai|essais"

# exclusions — un chemin, puis son motif, sur une ligne.
#
# Une exclusion répond à UNE question : ce fichier porte-t-il du français parce
# qu'il DÉFINIT la règle, ou parce qu'il la VIOLE ? Seul le premier cas
# s'exclut.
exclusions() {
    cat <<'FIN'
tools/verifie-langue-du-code.sh porte le vocabulaire qu'il refuse — c'est sa définition même
tools/testdata/langue-du-code témoin d'échec (ADR 013) — faux exprès, ne jamais « corriger »
FIN
}

# lignes_ajoutees rend les lignes `+` du diff, hors en-têtes, dans les zones de
# code, hors chemins exclus.
# ⚠️ `base` et `head` sont capturés AVANT `set --`, qui écrase les paramètres
# positionnels. La première version les lisait après : `git diff` recevait
# `:(exclude)…` en guise de révision, échouait sur « bad revision », et le
# `|| true` de fin transformait cet échec en « rien à signaler ».
#
# Un garde qui rend 0 quand son git échoue est un garde qui ne peut pas échouer.
# Trouvé en lisant la SORTIE, pas le code de retour — celui-ci était à 0.
lignes_ajoutees() {
    if [ -n "${LANGUE_LIGNES:-}" ]; then
        printf '%s\n' "$LANGUE_LIGNES"
        return 0
    fi

    base=$1
    head=$2

    # Le pathspec combine les zones et l'extension : `internal/**/*.go`, etc.
    set --
    for zone in $ZONES; do
        set -- "$@" ":(glob)$zone/**/$EXTENSIONS"
    done
    while IFS=' ' read -r chemin _; do
        [ -n "$chemin" ] || continue
        set -- "$@" ":(exclude)$chemin"
    done <<FIN
$(exclusions)
FIN

    # Le diff est capturé AVANT d'être filtré : dans un tube, l'échec de `git`
    # serait masqué par le succès du dernier `grep`.
    if ! diff_brut=$(git diff "$base" "$head" -- "$@"); then
        echo "  git diff a échoué sur '$base'..'$head' — le garde n'a rien pu lire" >&2
        return 1
    fi

    printf '%s\n' "$diff_brut" | grep -E '^\+' | grep -Ev '^\+\+\+' || true
}

# declarations ne garde que les lignes qui DÉCLARENT un identifiant Go, et vide
# les chaînes littérales qu'elles contiennent.
#
# Un identifiant se déclare, il ne s'invente pas au milieu d'une expression : ne
# regarder que les déclarations évite d'accuser l'appel d'une fonction d'une
# dépendance, et rend le garde silencieux sur tout ce qui n'est pas à nous.
#
# Le vidage n'est pas cosmétique. Sans lui, la ligne
#
#	if guard := read(t, filepath.Join(dir, "AMORCAGE.md")); ... {
#
# se faisait accuser pour le mot français d'un CHEMIN DE FICHIER. Un identifiant
# ne vit jamais entre guillemets ; ce qui s'y trouve est une donnée, et une
# donnée peut légitimement être en français — un nom de document, un message
# destiné à l'utilisateur (ADR 018, hors champ).
declarations() {
    grep -E '^\+[[:space:]]*(func|type|var|const)[[:space:]]|:=' \
        | sed -e 's/"[^"]*"/""/g' -e "s/\`[^\`]*\`/\`\`/g" || true
}

commentaires() {
    grep -E '^\+[[:space:]]*//' || true
}

# segmente coupe le camelCase en mots séparés, puis met en minuscules.
#
# `depilerLaFile` devient `depiler la file`, donc `depiler` est un mot ENTIER et
# `grep -w` le trouve. Sans cette étape, il faudrait `\b` ou `\<` — deux
# extensions que le grep de la toolbox (busybox) ne traite pas comme celui de la
# CI (GNU). Le dépôt a déjà payé une divergence de grep entre les deux : un
# motif qui passe sur l'hôte et ne cherche plus rien dans l'image.
#
# L'effet de bord est voulu : `decomposer` reste un seul mot, et
# n'est donc PAS accusé pour le `composer` qu'il contient.
segmente() {
    sed -e 's/\([a-z0-9]\)\([A-Z]\)/\1 \2/g' | tr '_' ' ' | tr '[:upper:]' '[:lower:]'
}

controle() {
    # Sans ce test, un échec de lecture du diff donnerait une liste vide, donc
    # un garde vert. C'est la forme exacte du faux vert que ce dépôt traque.
    if ! ajoutees=$(lignes_ajoutees "${1:-}" "${2:-}"); then
        return 1
    fi
    fail=0

    accentues=$(printf '%s\n' "$ajoutees" | declarations | grep -E "$MOTIF_ACCENT" || true)
    if [ -n "$accentues" ]; then
        echo "  identifiant Go portant une marque diacritique :" >&2
        printf '%s\n' "$accentues" | sed 's/^/    /' >&2
        fail=1
    fi

    francais=$(printf '%s\n' "$ajoutees" | declarations | segmente | grep -Ew "$MOTS" || true)
    if [ -n "$francais" ]; then
        echo "  identifiant Go puisant dans le vocabulaire français :" >&2
        printf '%s\n' "$francais" | sed 's/^/    /' >&2
        fail=1
    fi

    prose=$(printf '%s\n' "$ajoutees" | commentaires | grep -E "$MOTIF_ACCENT" || true)
    if [ -n "$prose" ]; then
        echo "  commentaire en français :" >&2
        printf '%s\n' "$prose" | sed 's/^/    /' >&2
        fail=1
    fi

    if [ "$fail" -ne 0 ]; then
        echo "" >&2
        echo "ERREUR: la langue du CODE est l'anglais — ADR 018." >&2
        echo "Le règlement, les ADR, les messages de commit et les PR restent en français." >&2
        echo "Les messages destinés à l'UTILISATEUR (domain.Error.Message) sont hors champ :" >&2
        echo "ils relèvent de l'internationalisation, issue #12." >&2
        return 1
    fi

    echo "  langue du code : rien à signaler sur les lignes ajoutées."
    return 0
}

# temoin prouve les DEUX moitiés du garde.
#
# Un garde qui refuse tout passerait les cas de refus et serait inutilisable ;
# un garde qui accepte tout passerait le dernier et ne garderait rien. Les
# quatre ensemble sont la seule preuve qu'il DISCRIMINE.
temoin() {
    echec=0
    _cas "identifiant accentue" 'func creerUtilisateurVérifié() {' 1 || echec=1
    _cas "identifiant francais sans accent" 'func depilerLaFile() error {' 1 || echec=1
    _cas "commentaire francais" '// Cette fonction rend une valeur normalisée.' 1 || echec=1
    _cas "anglais, declaration et commentaire" '// NewUser builds a user.
func NewUser(email string) User {' 0 || echec=1
    # Le cinquième cas garde le garde lui-même : une révision qui n'existe pas
    # doit REFUSER, pas rendre une liste vide déguisée en succès.
    _cas_git "revision inexistante" 1 || echec=1

    [ "$echec" -eq 0 ] && echo "  témoin complet : le garde refuse ET sait être satisfait"
    return "$echec"
}

# _cas_git éprouve le chemin où le garde LIT le dépôt, que `_cas` court-circuite
# par `LANGUE_LIGNES`. Sans lui, le défaut « bad revision rendu en succès » ne
# serait attrapé par aucun témoin.
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
    contenu=$(printf '%s\n' "$2" | sed 's/^/+/')
    attendu=$3

    obtenu=0
    LANGUE_LIGNES="$contenu" controle >/dev/null 2>&1 || obtenu=$?

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
