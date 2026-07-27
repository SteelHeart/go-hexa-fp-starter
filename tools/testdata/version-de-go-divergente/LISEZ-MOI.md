# Témoin — ces fichiers DOIVENT faire échouer le garde de version de Go

Ils sont **faux exprès** : `go.mod.temoin` déclare une version que les deux
images ne portent pas. Ce n'est pas un module Go — le suffixe `.temoin` existe
précisément pour qu'aucun outil de la chaîne Go ne les prenne pour de vrais
fichiers de configuration.

Raison d'être : l'[ADR 013](../../../documentation/adr/013-un-garde-doit-savoir-echouer.md).
Sans eux, « les trois versions concordent » et « le garde ne lit plus rien »
rendent exactement la même sortie.

C'est la situation qu'aurait créée la PR #63, qui proposait de monter la seule
image de livraison : l'équipe aurait développé sous une version et livré un
binaire compilé sous une autre.

⚠️ **Ne pas « corriger » ces fichiers.**
