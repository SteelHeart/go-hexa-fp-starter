#!/usr/bin/env sh
# Confronte la DOCUMENTATION au dépôt réel — issue #118, audit #107.
#
#   verifie-veracite-doc.sh            les trois contrôles
#   verifie-veracite-doc.sh --temoin   prouve qu'ils savent refuser ET être satisfaits
#
# ─────────────────────────────────────────────────────────────────────────────
# Pourquoi ce garde existe
# ─────────────────────────────────────────────────────────────────────────────
# L'audit #107 a trouvé douze écarts. ONZE portaient sur la documentation, un
# seul sur le code. Ce n'est pas un hasard : chaque règle de forme du code a son
# garde qui refuse, et AUCUNE affirmation de la documentation n'avait quoi que
# ce soit qui la confronte au dépôt.
#
# La dérive s'installe donc exactement là où rien ne regarde — et c'est la
# partie sur laquelle tout le monde s'appuie pour décider.
#
# ─────────────────────────────────────────────────────────────────────────────
# Ce que ce garde vérifie, et ce qu'il ne peut pas vérifier
# ─────────────────────────────────────────────────────────────────────────────
# Il vérifie des faits MÉCANIQUES : un chemin existe ou non, un pilote est
# déclaré ou non. Il ne juge pas la prose, et il ne saura jamais si une phrase
# est encore vraie.
#
# C'est assumé, et c'est déjà beaucoup : sur les douze écarts de l'audit, trois
# — É-02, É-04, É-05 — n'auraient pas pu exister. Les quatorze godoc menteurs de
# #127, eux, lui échappent entièrement.
#
# ─────────────────────────────────────────────────────────────────────────────
# Ce que la campagne de traduction a démontré, et qui a motivé ce garde
# ─────────────────────────────────────────────────────────────────────────────
# Le compteur de l'invariant « plus de deux retours = un type manquant » valait
# CINQ, SIX ou SEPT selon le fichier. Trois valeurs pour un même fait, et ce
# chiffre sert d'ARGUMENT.
#
# Un fait recopié à la main dérive. Toujours. La grille des personas annonçait
# six rouges et en portait sept ; le tableau de `pilotes.md` annonçait onze
# pilotes quand il y en avait quinze. Le remède n'est pas d'écrire mieux, c'est
# de RECOMPTER.
set -eu

AMORCAGE="documentation/AMORCAGE.md"
PILOTES="documentation/technique/pilotes.md"

echec=0

# ─────────────────────────────────────────────────────────────────────────────
# Contrôle 1 — la carte du dépôt contre `git ls-files`
# ─────────────────────────────────────────────────────────────────────────────
# Aurait rendu É-05 impossible : huit chemins hors carte, et un cartographié qui
# n'existait pas.
#
# La carte porte des chemins ET des motifs — `cmd/{server,worker}`,
# `config/*.yaml`, `internal/core/{nom}/`. Seuls les chemins LITTÉRAUX sont
# vérifiés : un motif ne désigne pas un chemin, et l'étendre reviendrait à
# réimplémenter un shell. C'est la limite, elle est écrite.
# Les lignes marquées ⟨absent⟩ sont ÉCARTÉES : la carte documente délibérément
# certaines absences — `api/openapi.yaml` n'existe pas, le contrat est servi sur
# `/openapi.{json,yaml}` et un fichier généré à la main dériverait. Refuser ces
# lignes ferait échouer le garde sur une vérité que la carte énonce.
chemins_cartographies() {
    sed -n '/^## Où se trouve quoi/,/^Huit modules noyau/p' "$AMORCAGE" \
        | grep -v '⟨absent⟩' \
        | grep -oE '^[A-Za-z._][A-Za-z0-9._/-]*/' \
        | sed 's/[[:space:]]*$//' \
        | sort -u
}

controle_carte() {
    echo "  1. la carte du dépôt contre git ls-files"

    manquants=$(
        chemins_cartographies | while read -r chemin; do
            [ -n "$chemin" ] || continue
            [ -e "$chemin" ] || echo "$chemin"
        done
    )
    if [ -n "$manquants" ]; then
        echo "     chemin CARTOGRAPHIÉ qui n'existe pas :" >&2
        printf '%s\n' "$manquants" | sed 's/^/       · /' >&2
        return 1
    fi

    # Le sens inverse : un dossier de premier niveau que la carte ignore. Le
    # contrôle s'arrête au premier niveau — descendre exigerait de décider quels
    # sous-dossiers méritent une ligne, ce qui est un jugement, pas un fait.
    hors_carte=$(
        git ls-files | awk -F/ 'NF>1 {print $1"/"}' | sort -u | while read -r dossier; do
            chemins_cartographies | grep -q "^$dossier" || echo "$dossier"
        done
    )
    if [ -n "$hors_carte" ]; then
        echo "     dossier de premier niveau ABSENT de la carte :" >&2
        printf '%s\n' "$hors_carte" | sed 's/^/       · /' >&2
        return 1
    fi

    echo "     tous les chemins cartographiés existent, et réciproquement."
    return 0
}

# ─────────────────────────────────────────────────────────────────────────────
# Contrôle 2 — le tableau de `pilotes.md` contre les `catalog.go`
# ─────────────────────────────────────────────────────────────────────────────
# Aurait rendu É-04 impossible : le document annonçait onze pilotes construits,
# il y en avait quinze, et trois modules n'y figuraient pas du tout.
#
# La source de vérité est le CODE : chaque module déclare ses pilotes dans son
# `catalog.go`, à côté de la fabrique, en partageant ses constantes avec elle
# (ADR 014). Le document en est une copie, et une copie dérive.
# Des paires `module:pilote`, jamais des noms de pilotes seuls.
#
# ⚠️ La première version comparait les noms DÉDUPLIQUÉS, et elle était trop
# faible pour attraper É-04, l'écart qu'elle existe pour empêcher : si `auth`
# disparaissait du tableau, `memory` y resterait annoncé par `outbox`, et le
# garde passerait. C'est précisément ce qui s'était produit — trois modules
# absents du tableau, tous leurs pilotes portant des noms déjà cités ailleurs.
pilotes_declares() {
    for fichier in internal/core/*/*.go internal/modules/*/*.go; do
        [ -f "$fichier" ] || continue
        module=$(basename "$(dirname "$fichier")")
        # `[Dd]river` : les modules NOYAU nomment leurs constantes
        # `driverMemory`, non exportées ; `user_registration` exporte
        # `DriverMemory`. Divergence réelle du dépôt, trouvée par ce garde à sa
        # première exécution — il ne voyait qu'une moitié des pilotes.
        grep -ohE '[Dd]river[A-Za-z]+ +=? *"[a-z-]+"' "$fichier" 2>/dev/null \
            | sed "s/.*\"\(.*\)\"/$module:\1/"
    done | sort -u
}

# SEULE la deuxième colonne est lue. La première porte le nom du module, dans
# des accents graves elle aussi : la prendre reviendrait à comparer des modules
# à des pilotes, et le garde accuserait `auth` d'être un pilote inexistant.
#
# Ce défaut était dans la première version, et c'est le contrôle lui-même qui l'a
# montré à sa première exécution — il accusait les huit modules.
pilotes_annonces() {
    sed -n '/^| Module | Pilotes construits/,/^$/p' "$PILOTES" \
        | grep '^|' \
        | awk -F'|' 'NF>2 {
              module = $2; pilotes = $3
              # Le nom du module est le PREMIER groupe entre accents graves de
              # la colonne : la cellule peut porter une glose — « *(module
              # métier)* » — que le prendre en entier collerait au nom.
              if (match(module, /`[a-z_]+`/) == 0) next
              module = substr(module, RSTART + 1, RLENGTH - 2)
              n = split(pilotes, morceaux, "·")
              for (i = 1; i <= n; i++) {
                  pilote = morceaux[i]
                  if (match(pilote, /`[a-z-]+`/) == 0) continue
                  print module ":" substr(pilote, RSTART + 1, RLENGTH - 2)
              }
          }' | sort -u
}

controle_pilotes() {
    echo "  2. le tableau de pilotes.md contre les catalog.go"

    declares=$(pilotes_declares)
    annonces=$(pilotes_annonces)

    if [ -z "$declares" ]; then
        echo "     aucun pilote trouvé dans le code — le motif ne cherche plus rien" >&2
        return 1
    fi
    if [ -z "$annonces" ]; then
        echo "     aucun pilote trouvé dans $PILOTES — le tableau a changé de forme" >&2
        return 1
    fi

    # Fichiers temporaires plutôt que substitution de processus : `<(…)` est du
    # bash, et ces gardes tournent sous le `sh` de la toolbox comme sous celui
    # de la CI. Le dépôt a déjà payé une divergence de shell entre les deux.
    tmp_declares=$(mktemp)
    tmp_annonces=$(mktemp)
    printf '%s\n' "$declares" > "$tmp_declares"
    printf '%s\n' "$annonces" > "$tmp_annonces"

    oublies=$(grep -Fxv -f "$tmp_annonces" "$tmp_declares" || true)
    inventes=$(grep -Fxv -f "$tmp_declares" "$tmp_annonces" || true)
    rm -f "$tmp_declares" "$tmp_annonces"

    if [ -n "$oublies" ]; then
        echo "     pilote DÉCLARÉ dans un catalog.go et absent du tableau :" >&2
        printf '%s\n' "$oublies" | sed 's/^/       · /' >&2
        return 1
    fi

    if [ -n "$inventes" ]; then
        echo "     pilote ANNONCÉ dans le tableau et déclaré par aucun catalog.go :" >&2
        printf '%s\n' "$inventes" | sed 's/^/       · /' >&2
        return 1
    fi

    echo "     $(printf '%s\n' "$declares" | wc -l | tr -d ' ') paires module:pilote déclarées, toutes annoncées, et réciproquement."
    return 0
}

# ─────────────────────────────────────────────────────────────────────────────
# Contrôle 3 — l'anatomie de l'ADR 012 contre l'arborescence réelle
# ─────────────────────────────────────────────────────────────────────────────
# Aurait fait sauter É-02 et É-09 au premier passage, des mois plus tôt.
#
# Ce contrôle est resté minimal tant que #116 n'était pas tranchée : l'ADR 012
# gravait `surfaces/`, le code écrivait `adapters/primary/`, et trancher dans un
# garde reviendrait à décider par l'outillage ce qu'un ADR doit décider.
#
# L'ADR 019 a tranché — `adapters/{primary,secondary}` — donc le garde peut
# désormais l'appliquer : AUCUN dossier `surfaces/` ne doit réapparaître.
#
# `application/` reste ABSENT de la liste des dossiers exigés : la dérogation
# É-09 est écrite dans l'ADR 019 §3, quatre modules noyau n'ont aucune politique
# à orchestrer, et une couche qui réexporterait le pilote serait la couche de
# transfert que l'hexagonal reproche aux couches de service.
controle_anatomie() {
    echo "  3. l'anatomie ADR 012 + 019 contre l'arborescence"

    manque=$(
        for module in internal/core/*/ internal/modules/*/; do
            [ -d "$module" ] || continue
            nom=$(basename "$module")
            # `internal/core/tests` porte les tests TRANSVERSES aux modules, ce
            # n'est pas un module. Le nom suffit à le distinguer, et une liste
            # d'exceptions plus large finirait par couvrir un vrai oubli.
            [ "$nom" = "tests" ] && continue
            for exige in domain ports tests; do
                [ -d "$module$exige" ] || echo "$module manque $exige/"
            done
            for exige in module.go catalog.go; do
                [ -f "$module$exige" ] || echo "$module manque $exige"
            done
            # ADR 019 : le nom retenu est `adapters/`, jamais `surfaces/`. Ce
            # contrôle empêche la réapparition silencieuse de l'ancien nom —
            # c'est-à-dire le retour de l'écart É-02, qui a vécu des mois.
            [ -d "${module}surfaces" ] && echo "$module porte surfaces/ — l'ADR 019 impose adapters/"
        done
    )
    if [ -n "$manque" ]; then
        echo "     module hors anatomie ADR 012 :" >&2
        printf '%s\n' "$manque" | sed 's/^/       · /' >&2
        return 1
    fi

    echo "     tous les modules portent domain/ ports/ tests/ module.go catalog.go."
    return 0
}

# ─────────────────────────────────────────────────────────────────────────────
# Mode témoin — les trois contrôles savent-ils encore refuser ?
# ─────────────────────────────────────────────────────────────────────────────
# Chaque contrôle a ses DEUX cas : il refuse un dépôt fautif, et il est
# satisfait par le dépôt réel. Sans le second, un contrôle cassé et un contrôle
# sévère sont indiscernables — la leçon du volet « noms de fichiers » du garde
# de mention, qui passait son cas de refus parce que sa fonction n'existait pas.
temoin() {
    echec_temoin=0
    temoin_dir="tools/testdata/veracite-doc"

    if [ ! -d "$temoin_dir" ]; then
        echo "  le témoin $temoin_dir a disparu : le garde n'est plus vérifiable" >&2
        exit 1
    fi

    # Cas de REFUS : la carte du témoin nomme un chemin qui n'existe pas.
    obtenu=0
    AMORCAGE="$temoin_dir/carte-fautive.md" controle_carte >/dev/null 2>&1 || obtenu=$?
    _verdict "carte : un chemin cartographié absent" 1 "$obtenu" || echec_temoin=1

    # Cas de REFUS : le tableau du témoin annonce un pilote qui n'existe pas.
    obtenu=0
    PILOTES="$temoin_dir/pilotes-fautif.md" controle_pilotes >/dev/null 2>&1 || obtenu=$?
    _verdict "pilotes : un pilote annoncé et non déclaré" 1 "$obtenu" || echec_temoin=1

    # Cas d'ACCEPTATION : le dépôt réel. C'est la moitié sans laquelle un garde
    # qui refuserait tout passerait les deux cas ci-dessus.
    for controle in controle_carte controle_pilotes controle_anatomie; do
        obtenu=0
        "$controle" >/dev/null 2>&1 || obtenu=$?
        _verdict "$controle sur le dépôt réel" 0 "$obtenu" || echec_temoin=1
    done

    [ "$echec_temoin" -eq 0 ] && echo "  témoin complet : les trois refusent ET savent être satisfaits"
    return "$echec_temoin"
}

_verdict() {
    nom=$1
    attendu=$2
    obtenu=$3

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

controle_carte || echec=1
controle_pilotes || echec=1
controle_anatomie || echec=1

if [ "$echec" -eq 0 ]; then
    echo "  la documentation dit vrai sur ce que ce garde sait vérifier."
fi
exit "$echec"
