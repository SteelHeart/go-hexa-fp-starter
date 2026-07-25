# Journal de friction

Ce que le socle ne fournit pas et qui fait perdre du temps, ou pire, qui pousse à contourner une
règle. Une friction non écrite se paie à chaque fois qu'elle se produit, sans jamais devenir
visible.

**Format** : `F{NNN}` · date · ce qui bloque · ce que ça coûte · statut.
Une friction ouverte a son issue, avec le label `friction`.

| Réf | Date | Friction | Coût | Statut |
|---|---|---|---|---|
| F001 | 2026-07-25 | Docker absent de la machine de développement de référence | Les tests d'intégration, de bout en bout et les migrations ne sont pas exécutables en local. La CI est le seul endroit où ils tournent, donc la boucle de retour sur le SQL est de plusieurs minutes | **Ouvert** — assumé : le cœur reste vérifiable en local, et c'est ce qui rend la contrainte tenable |
| F002 | 2026-07-25 | La protection de branche serveur exige un plan payant sur dépôt privé | Le crochet `pre-push` est un filet contournable avec `--no-verify`, pas un contrôle | **Ouvert** — arbitrage issue #18 : dépôt public, GitHub Pro, ou assumer l'absence |
| F003 | 2026-07-25 | Aucun test de mutation | La couverture mesure ce qui est **exécuté**, pas ce qui est **vérifié** : 90 % sur le cœur ne prouve pas que les assertions sont justes | **Ouvert** |
| F004 | 2026-07-25 | Les versions d'outillage sont en `latest` en CI | Un linter qui change de comportement rend la CI non reproductible et peut casser une PR sans rapport | **Ouvert** — à figer à la première release ([`rules/dependances.md`](../../rules/dependances.md) §5) |
| F005 | 2026-07-25 | `-race` exige CGO et un compilateur C, absents de la machine de référence Windows | Le détecteur de courses ne tourne qu'en CI. Un défaut de concurrence introduit en local n'est vu qu'à la PR | **Ouvert** — assumé : `task test` sans `-race` en local, `task test:race` en CI Linux |
| F006 | 2026-07-25 | `task` (go-task) et `govulncheck` ne sont pas installés sur la machine de référence | `task check` n'a **jamais** été exécuté tel quel en local : les étapes sont lancées une par une, donc rien ne garantit que l'enchaînement est le même qu'en CI. C'est un faux vert en puissance — la barrière qu'on croit franchir n'est pas celle qui tourne | **Ouvert** — `task init` est censé les poser ; vérifier pourquoi il ne l'a pas fait |
| F007 | 2026-07-25 | Chaîne d'outils Go en 1.25.4, **20 vulnérabilités de la bibliothèque standard** atteignables | `govulncheck` échoue, donc `task check` ne peut pas être vert. Toutes sont corrigées par Go ≥ **1.25.12** : `crypto/tls`, `crypto/x509`, `net/url`, `net/mail`, `html/template`, `os`. Aucune ne vient d'une dépendance du dépôt | **Ouvert** — monter la chaîne d'outils Go. **Bloque le tag `v0.1.0` (#1)** |

## Quand écrire une entrée

- On a contourné une règle, même proprement, parce que l'outillage ne permettait pas de la suivre.
- On a passé plus de trente minutes sur un problème qui n'a rien à voir avec le métier.
- On s'est dit « il faudra qu'on regarde ça un jour ».

Ce journal n'est pas une liste de tâches : c'est le relevé des endroits où le cadre coûte plus
qu'il ne rapporte. C'est ce qui permet de le corriger plutôt que de le subir.
