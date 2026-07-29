# ADR 020 — La licence est Apache-2.0, et l'irréversibilité commence ici

- **Statut** : acceptée
- **Date** : 2026-07-29
- **Issue** : [#155](https://github.com/SteelHeart/go-hexa-fp-starter/issues/155)
- **Inverse** : l'arbitrage rendu sur [#113](https://github.com/SteelHeart/go-hexa-fp-starter/issues/113)
  — *source-available assumé*. Cette décision-là n'avait produit **aucun ADR** : elle vivait dans un
  commentaire d'issue et dans le `README`, ce que la hiérarchie des sources ne prévoit pas pour une
  zone à cette inertie. C'est corrigé par le présent document.

## Contexte

Le dépôt est **public en lecture** depuis le 2026-07-28, sous *tous droits réservés*. Cet état a été
choisi, écrit, et assumé (#113, PR #120) : lisible par tout le monde, utilisable par personne.

### Ce qui a rendu la question exigible

Le transfert vers l'organisation est subordonné à une condition explicite : les modules de base de
la v0.1.0 prêts **et éprouvés du point de vue de toutes les personas**.

Les preuves ont été mesurées le 2026-07-29 ([#138](https://github.com/SteelHeart/go-hexa-fp-starter/issues/138)),
et **P3 — l'équipe qui adopte de l'extérieur — a conclu par un avertissement** :

> **P3 restera rouge tant que la licence n'aura pas changé, quel que soit l'état du code.** C'est
> écrit ici pour que personne n'entreprenne la frontière publique en croyant débloquer cette
> persona.

P3 veut **dépendre** du socle plutôt que le copier. Sous *tous droits réservés*, elle n'en a pas le
droit — même une frontière publique livrée, même zéro `internal/`, même un tag `v1.0.0`. Elle était
rouge **par décision**, pas par défaut.

**La condition de transfert et l'arbitrage de #113 se contredisaient donc**, et la contradiction
n'était visible nulle part avant que les personas ne soient mesurées plutôt qu'affirmées. C'est ce
que ce dossier de preuves existe pour produire.

La contradiction se lève d'un côté ou de l'autre : amender la condition en excluant P3 avec son
motif, ou ouvrir la licence. **Le PO a tranché pour l'ouverture.**

## Décision

### 1. La licence est **Apache-2.0**

Ce n'est pas un choix neuf. Il était **déjà écrit comme recommandation non tranchée** à trois
endroits — l'ancien `LICENSE`, le commentaire d'arbitrage de #113, et la grille des personas — et
toujours pour la même raison.

| Motif | Détail |
|---|---|
| **Concession de brevet explicite** (§3) | La seule des candidates courantes à en porter une. Décisif pour un socle **destiné à être intégré dans des produits tiers** : sans elle, l'adoptant s'expose à une revendication de brevet du contributeur dont il vient d'intégrer le code, et il n'a aucun moyen d'évaluer ce risque lui-même |
| **Contributions entrantes réglées par la licence** (§5) | *« Unless You explicitly state otherwise, any Contribution […] shall be under the terms and conditions of this License. »* Entrant = sortant, **sans CLA ni DCO à écrire**. #113 posait la gouvernance des contributions comme une question ouverte à trancher séparément : Apache-2.0 est la seule candidate qui y répond seule |
| **Attribution par `NOTICE`** (§4d) | Traçabilité de la paternité, sans copyleft |
| **Représailles de brevet** (§3, seconde phrase) | Qui attaque en brevet perd la licence. Protection réciproque, pas seulement descendante |

**Pourquoi pas MIT.** Plus courte, mieux connue, et **muette sur les brevets**. Sur une application
c'est acceptable ; sur un framework que des tiers intègrent, cela transfère à l'adoptant un risque
qu'il ne peut pas mesurer.

**Pourquoi pas MPL-2.0 ni un copyleft.** Le copyleft par fichier obligerait tout adoptant à publier
ses modifications du socle. C'est précisément ce que P3 refuse — et l'ouvrir pour P3 sous une
licence que P3 refuse n'aurait aucun sens.

**Pourquoi pas BUSL ou PolyForm.** Elles étaient les candidates cohérentes de la posture
*source-available*, qui vient d'être abandonnée. Elles laissent P3 rouge.

### 2. Pas d'en-tête de licence par fichier

L'annexe d'Apache-2.0 *suggère* un préambule de onze lignes par fichier. **Décision : non**, et
c'est écrit ici pour que la question ne se redemande pas à chaque relecture.

- L'écosystème Go ne le pratique pas — la bibliothèque standard elle-même porte trois lignes, pas
  onze, et une grande partie des modules publics n'en portent aucune ;
- `LICENSE` et `NOTICE` à la racine suffisent : §4(a) et §4(d) portent sur la **distribution**, pas
  sur chaque fichier ;
- 169 fichiers de production, chacun préfixé de onze lignes, c'est **1 859 lignes** dont aucune ne
  se relit — et le dépôt refuse par principe le texte que personne ne lit.

### 3. `documentation/`, `rules/` et les ADR sont couverts, et restent en français

Apache-2.0 définit « Source » comme incluant *« documentation source, and configuration files »*.
Le règlement et les ADR sont donc distribués sous la même licence — ce qui est voulu : **un socle
dont on ne peut pas lire les règles n'est pas adoptable.**

L'[ADR 018](018-la-langue-du-code-est-l-anglais.md) reste en vigueur : le code en anglais, `rules/`
en français. L'ouverture ne la remet pas en cause, mais elle **augmente son coût** — P5 mesure
4 917 lignes de coût d'entrée, dont 4 593 en français. Ce coût cesse d'être interne le jour où
l'adoption externe devient l'objectif. À réexaminer, pas dans ce lot.

## Conséquences

### 🔴 L'irréversibilité commence aujourd'hui

**Une version publiée sous Apache-2.0 le reste.** À compter de la fusion de ce lot, quiconque a
récupéré ce commit peut l'utiliser, le modifier et le redistribuer sous ces termes, **définitivement**.
Un changement de licence ultérieur ne vaudrait que pour les versions suivantes ; il ne rappellerait
rien.

L'[ADR 015](015-la-frontiere-publique-est-derivee-d-un-usage-mesure.md) écrivait, dans sa colonne
« ce qui a été écarté » :

> *« la frontière est urgente » — le dépôt est **privé**, et le reste (LICENSE). Rien n'est publié,
> donc rien n'est figé : **l'irréversibilité n'a pas commencé**.*

Cette phrase était vraie le jour où elle a été écrite. Elle a cessé de l'être une première fois le
2026-07-28 (passage en public), et complètement aujourd'hui.

**Ce n'est PAS un argument contre l'ADR 015 — c'est ce qui le rend critique.** Sa règle — *la
frontière publique se dérive d'un usage mesuré, elle ne se décrète pas* — passe de prudente à
indispensable :

> **Tout paquet rendu importable devient utilisable pour toujours. Un paquet publié par excès ne se
> retire plus.**

L'ADR 015 le disait déjà — *« l'erreur irait dans le sens coûteux »* — en supposant que le coût
restait théorique. Il ne l'est plus.

### Ce que ce lot NE fait PAS

**Il ne fait pas passer P3 au vert.** Il retire le blocage qui rendait le travail inutile. Trois
critères restent rouges, et chacun est un lot à part entière :

| Critère de P3 | État au 2026-07-29 |
|---|---|
| Paquets importables depuis l'extérieur | **0** — les cinq paquets hors `internal/` sont tous `package main`. Il faut en **créer**, pas en promouvoir |
| Politique de versions et de dépréciation | **inexistante** — zéro tag, aucun `CHANGELOG`. Dépend de [#89](https://github.com/SteelHeart/go-hexa-fp-starter/issues/89) |
| Frontière API publique / interne | **inexistante** — l'ADR 015 en fixe la **méthode**, pas le résultat |

Le critère « Licence » lui-même est **reformulé**. La preuve P3 avait relevé que *« licence :
existe »* ne mesure pas ce qui compte : le critère était formellement atteint alors que P3 n'avait
le droit de rien faire. Il devient **« licence permettant l'usage »**, et passe au vert pour la
bonne raison.

### Ce qui change dans le dépôt

- `LICENSE` — texte Apache-2.0 **intégral**, jamais paraphrasé ;
- `NOTICE` — attribution (§4d), et le rappel que les dépendances ne sont pas vendorées ;
- `Dockerfile` — `org.opencontainers.image.licenses` passe de `NONE` à `Apache-2.0`. ⚠️ **Cette
  étiquette a déjà menti** : elle annonçait `MIT` sans qu'aucune licence MIT n'ait été accordée
  (#61). Elle doit dire exactement ce que dit `LICENSE`, et rien d'autre ;
- `README.md` — l'avertissement de tête et la section « Licence » ;
- la grille des personas et la preuve P3.

### Ce qui reste à écrire, et n'est pas dans ce lot

`CONTRIBUTING.md` et `CHANGELOG.md`. Apache-2.0 §5 règle la question **juridique** des contributions
entrantes ; il ne dit rien du **processus** — qui relit, sous quelles règles, avec quelle politique
de version. Lot distinct, et il n'est pas bloquant : une contribution reçue aujourd'hui est déjà
sous licence claire.

## Ce qui a été écarté

| Option | Pourquoi non |
|---|---|
| **Amender la condition de transfert** en excluant P3, motif écrit | Cohérent, et c'était l'autre issue honnête. Écarté par le PO : P3 est la persona de l'**adoption externe**, et un framework transféré à une organisation pour être adopté ne peut pas exclure la persona qui l'adopte |
| **MIT** | Aucune clause de brevets. Transfère à l'adoptant un risque qu'il ne peut pas évaluer |
| **MPL-2.0 / copyleft** | Oblige P3 à publier ses modifications — ce qu'elle refuse. Ouvrir pour P3 sous une licence que P3 refuse n'a pas de sens |
| **BUSL / PolyForm** | Candidates cohérentes de la posture *source-available*, qui vient d'être abandonnée. Laissent P3 rouge |
| **Attendre le tag `v0.1.0`** | La licence détermine ce que le transfert et le tag **signifient**. Elle se tranche avant, comme #113 l'écrivait déjà |
| **En-têtes par fichier** | 1 859 lignes que personne ne lit, pour une obligation qui porte sur la distribution et non sur le fichier |
