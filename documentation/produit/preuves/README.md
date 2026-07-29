# Preuves par persona — issue #138

Condition de transfert : **avant tout transfert vers une organisation, chaque persona doit avoir une
preuve tangible qu'elle peut utiliser le framework pour son projet, selon SES critères.**

Cinq mini-projets de liste de tâches, un par persona, construits avec `hexa new` et
`hexa make:feature`, puis mesurés contre le tableau de critères de leur propre persona.

## Les règles de l'exercice

**Le même domaine pour les cinq**, délibérément. Ce qui est comparé n'est pas la difficulté du
métier, c'est **ce que le socle rend possible ou impossible à chaque persona**. Un domaine trivial
isole la variable.

**Cinq projets et non un seul**, parce que les critères ne se recouvrent pas. Le projet qui satisfait
P1 ne dit rien de P3 : P1 copie le socle, P3 veut en dépendre.

**Ce ne sont pas des démonstrations.** Un projet écrit pour flatter le socle ne mesure rien. Chaque
projet part des exigences de sa persona, pas de ce que le socle sait faire — et quand les deux
divergent, **c'est la divergence qui est le résultat**.

## Ce que chaque preuve contient

- le **journal de bord** : chaque commande, sa sortie, son code de retour — pas un résumé ;
- le **tableau de critères de la persona, remesuré**, ligne à ligne ;
- ce qu'il a fallu **écrire à la main** que le socle aurait dû fournir ;
- ce qu'il a fallu **contourner**, et si le contournement viole une règle.

## État

| Persona | Preuve | Verdict |
|---|---|---|
| [P1 — l'équipe produit](p1-equipe-produit.md) · **PRIMAIRE** | ✅ écrite | ✅ **trois critères sur quatre**, et le quatrième n'est plus celui qui était écrit |
| [P3 — l'adoption externe](p3-adoption-externe.md) | ✅ écrite | 🔴 **impossible en l'état** — et c'est mesuré, plus seulement affirmé |
| P2 — le fort trafic | à écrire | — |
| P4 — l'exploitant | à écrire | — |
| P5 — le décideur | à écrire | — |

⚠️ **Deux verdicts négatifs sont attendus, et ils comptent autant que les positifs.** La grille des
personas affirme des rouges depuis trois relevés sans les avoir jamais exercés. Un projet qui bute
est la première mesure réelle.
