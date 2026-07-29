# P5 — Le décideur technique · *ne code pas*

> **Verdict : ⚠️ trois critères sur quatre.** Le quatrième — le coût d'entrée — reste rouge, et il
> est désormais **chiffré**.

Mesuré le **2026-07-29**, sur `main`.

## Ce que P5 voulait savoir

Pas écrire du code : décider. *Combien de temps faut-il à quelqu'un qui découvre ce dépôt pour livrer
sa première tâche, et qu'a-t-il dû lire ?*

## Le tableau de critères de P5, remesuré

| Critère | Cible | Mesuré | Verdict |
|---|---|---|---|
| La barrière qualité a **réellement tourné** | oui | **oui** — `task check` vert de bout en bout, code de retour vérifié, et **20/20 vert en CI** | ✅ |
| Le socle dépend-il d'une personne ? | non | **non** — aucun `CODEOWNERS`, aucun pseudo dans les règles. Le chemin de module est la seule valeur nominative | ✅ |
| Documentation en accord avec l'état réel | oui | **oui, et c'est désormais OUTILLÉ** — voir ci-dessous | ✅ |
| Coût d'entrée d'un développeur | faible | **élevé** — **4 917 lignes** à lire, dont **4 593 en français** | 🔴 |

## Le critère qui a changé de nature

Le troisième était marqué *« ⚠️ à surveiller — le critère n'est pas acquis une fois pour toutes, il
se re-mesure »*. Il l'était à raison : l'audit #107 a trouvé **douze écarts, dont onze sur la
documentation**.

Depuis, **`tools/verifie-veracite-doc.sh` tourne à chaque PR** (#118) : la carte du dépôt contre
`git ls-files`, le tableau des pilotes contre les `catalog.go`, l'anatomie contre l'arborescence.

Pour P5, la différence est de nature : le critère passe de *« il faut y repenser »* à *« la CI le
refuse »*. C'est exactement ce qu'elle achète en choisissant ce socle.

⚠️ **Sa limite est écrite** : le garde ne juge pas la prose. Les **quatorze godoc qui décrivaient
l'inverse du code** (#127) lui échappaient entièrement — ils ont été trouvés en traduisant, pas par
un outil.

## Le coût d'entrée, chiffré

| Corpus | Lignes | Langue |
|---|---|---|
| `rules/` — 16 fichiers | 1 822 | français |
| Les 19 ADR | 2 006 | français |
| `documentation/AMORCAGE.md` | 765 | français |
| `README.md` | 324 | **anglais** |
| **Total** | **4 917** | dont **4 593 en français** |

**C'est le chiffre que P5 doit avoir en main**, et il n'existait pas avant cette preuve.

Trois lectures possibles, et elles ne s'excluent pas :

**La densité est la valeur.** Ces 4 917 lignes sont ce qui empêche la forme de se dégrader — c'est la
proposition même du socle, et un dépôt qui n'aurait rien à lire n'aurait rien à tenir.

**La densité est la barrière.** Un développeur qui doit lire 4 917 lignes avant sa première tâche ne
les lira pas. Il lira le README, copiera `user_registration`, et découvrira les règles **par les
refus de la CI** — ce qui fonctionne, mais coûte à chaque refus.

**La langue est un multiplicateur.** Le code est en anglais depuis l'ADR 018, donc lisible par tous.
Le règlement ne l'est pas. Pour une équipe non francophone, ces 4 593 lignes sont **inaccessibles**,
et le socle redevient un dépôt qu'on copie sans comprendre pourquoi il refuse.

## Ce qui n'a pas été mesuré, et qui devrait l'être

**Le délai réel avant la première tâche livrée par quelqu'un qui découvre.** Ce document mesure le
**volume à lire**, pas le temps qu'il faut. La différence compte : personne ne lit tout, et ce qui
importe est ce qu'il faut avoir lu pour ne pas se faire refuser.

Une mesure honnête exigerait quelqu'un qui découvre réellement le dépôt. **Je ne peux pas la
produire** : j'ai construit une partie de ce qui est mesuré, donc je ne découvre rien.

C'est écrit ici plutôt que remplacé par une estimation. Une estimation aurait l'air d'une mesure.

## Les deux rouges permanents ont disparu

La grille avertissait : *« Deux contrôles restent rouges en permanence, et il faut le dire à P5 avant
qu'elle ne le découvre »* — CodeQL (#72) et Deploy UAT (#75).

**Les deux sont réglés.** CodeQL s'exécute depuis le passage en public : **succès, 0 alerte**. Ce qui
comptait pour P5 n'était pas le job, c'était le principe : *un rouge permanent apprend à ignorer le
rouge.* Le principe a tenu — le job n'a jamais été rendu « non bloquant », et c'est ce qui a permis
de constater le changement le jour même.

## Ce que P5 doit savoir avant d'arbitrer

- **Le socle est tenu par des gardes, pas par de la discipline.** 21 règles `arch-go`, ~50
  analyseurs, huit gardes maison portant chacun le cas qui les fait échouer (ADR 013). Ils ont trouvé
  ce qu'aucune relecture n'a trouvé : une course de données, quatorze godoc menteurs, une clé de
  configuration sans effet.
- **Il n'est pas encore adoptable de l'extérieur.** Zéro paquet importable, licence tous droits
  réservés — voir [P3](p3-adoption-externe.md).
- **Une persona n'a reçu aucun gain en cinq relevés** — voir [P2](p2-fort-trafic.md). Ce n'est pas un
  oubli technique, c'est un ordre de priorité que personne n'a explicitement choisi.

*« Ce qu'elle tue : le règlement s'il devient un obstacle au recrutement. C'est la tension centrale
du projet. »* Elle n'est pas résolue. Elle est, pour la première fois, **chiffrée**.
