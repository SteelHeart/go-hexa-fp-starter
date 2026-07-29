# Témoins d'échec du garde de véracité — ADR 013

Deux documents **délibérément faux**. Ils font échouer `tools/verifie-veracite-doc.sh`, et c'est
**le but**.

- `carte-fautive.md` cartographie un chemin qui n'existe pas ;
- `pilotes-fautif.md` annonce un pilote qu'aucun `catalog.go` ne déclare.

**Ne jamais les « corriger ».** Sans eux, on ne distingue plus « la documentation dit vrai » de
« le garde ne cherche plus rien » — la confusion exacte qui a laissé onze gardes défectueux passer
inaperçus dans ce dépôt.

Le mode `--temoin` les éprouve, **et éprouve aussi le dépôt réel** : un garde qui refuserait tout
passerait les deux cas ci-dessus et serait inutilisable. Les deux moitiés ensemble sont la seule
preuve qu'il discrimine.
