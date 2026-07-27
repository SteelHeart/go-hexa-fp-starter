# Témoins — ces fichiers DOIVENT faire réagir le garde

Ils ne sont pas des artefacts du dépôt : ils sont **faux exprès**, et le garde
`tools/verifie-mention-outillage.sh` les exclut de son analyse du diff.

Leur raison d'être est l'[ADR 013](../../../documentation/adr/013-un-garde-doit-savoir-echouer.md) :
un garde est livré avec le cas qui le fait échouer. Sans eux, on ne distingue
pas « aucune mention dans le diff » de « le motif ne reconnaît plus rien » —
et un garde qui ne trouve rien ressemble en tout point à un garde satisfait.

**Un fichier par forme du motif.** Si une seule forme cesse d'être reconnue,
`--temoin` nomme laquelle, au lieu de dire seulement que le témoin passe.

| Fichier | Forme couverte | Pourquoi cette forme |
|---|---|---|
| `trailer-co-authored-by.txt` | pied de message de commit | la forme posée par défaut par l'outillage — celle qu'on n'écrit pas soi-même |
| `mention-generated-with.txt` | corps de PR | la formule posée en fin de description de PR |
| `emoji-robot.txt` | emoji seul | déjà cassée une fois par un ré-encodage Windows, sans que rien ne le signale |
| `adresse-noreply.txt` | adresse de courriel | passe sous le radar d'une relecture humaine : personne ne lit une adresse |

⚠️ **Ne pas « corriger » ces fichiers.** Les vider, c'est désarmer le garde.
