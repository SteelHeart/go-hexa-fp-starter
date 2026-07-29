# Témoin d'échec du garde de langue — ADR 013

Ce dossier contient du **français délibéré** dans du code Go. Il fait échouer
`tools/verifie-langue-du-code.sh`, et c'est **le but**.

**Ne jamais « corriger » ces fichiers.** Sans eux, on ne distingue plus « aucun français dans la PR »
de « le garde ne reconnaît plus rien » — la confusion exacte qui a laissé onze gardes défectueux
passer inaperçus dans ce dépôt.

Le mode `--temoin` du garde n'utilise pas ces fichiers : il éprouve le motif sur des lignes qu'il
fabrique. Ceux-ci servent à éprouver le garde **de bout en bout**, sur un vrai diff.
