# Témoin d'échec du garde de dette — ADR 013

Ce dossier porte des **marqueurs de dette délibérés**. Il fait échouer
`tools/verifie-dette.sh` quand on le lui donne à lire, et c'est **le but**.

**Ne jamais « corriger » ces fichiers.** Sans eux, on ne distingue plus « aucun marqueur dans la
PR » de « le motif ne reconnaît plus rien » — la confusion exacte qui a laissé onze gardes
défectueux passer inaperçus dans ce dépôt.

Le mode `--temoin` du garde n'utilise pas ces fichiers : il éprouve le motif sur des lignes qu'il
fabrique, ce qui lui permet de nommer LEQUEL des quatre marqueurs a cessé d'être reconnu. Ceux-ci
servent au contrôle de bout en bout, sur un vrai diff.
