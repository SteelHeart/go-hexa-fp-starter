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
| [P1 — l'équipe produit](p1-equipe-produit.md) · **PRIMAIRE** | ✅ | ✅ **trois critères sur quatre**, et le quatrième n'est plus celui qui était écrit |
| [P2 — le fort trafic](p2-fort-trafic.md) | ✅ | 🔴 **quatre sur cinq rouges** — dont un **pire** que la grille ne le disait (#141) |
| [P3 — l'adoption externe](p3-adoption-externe.md) | ✅ | 🔴 **impossible en l'état** — zéro paquet importable, mesuré. ⚠️ Cette preuve a **fait changer la licence** : elle a montré que le rouge était par décision, donc insensible à tout travail (ADR 020, #155) |
| [P4 — l'exploitant](p4-exploitant.md) | ✅ | ✅ **cinq sur cinq** — la seule persona entièrement verte |
| [P5 — le décideur](p5-decideur.md) | ✅ | ⚠️ **trois sur quatre** — le coût d'entrée est enfin **chiffré** : 4 917 lignes |

**Les cinq preuves sont écrites.** Deux verts, un mitigé, deux rouges — et les deux rouges étaient
annoncés avant d'être mesurés, ce qui est la seule façon qu'ils comptent comme mesure et non comme
surprise.

### Ce que l'exercice a trouvé, et que la grille ne pouvait pas voir

| Trouvaille | Persona | Suite |
|---|---|---|
| `max_body_bytes` **ne produit aucun effet** — une clé qui existe, est validée, et n'agit pas | P2 | [#141](https://github.com/SteelHeart/go-hexa-fp-starter/issues/141) |
| Les cinq paquets hors `internal/` sont des `package main` : **zéro** importable, pas « peu » | P3 | alimente l'ADR 015 |
| Le critère « 0 fichier du framework » doit se lire **« à modifier À LA MAIN »** | P1 | grille corrigée |
| Le coût d'entrée vaut **4 917 lignes, dont 4 593 en français** | P5 | chiffré pour la première fois |

⚠️ **Deux verdicts négatifs sont attendus, et ils comptent autant que les positifs.** La grille des
personas affirme des rouges depuis trois relevés sans les avoir jamais exercés. Un projet qui bute
est la première mesure réelle.
