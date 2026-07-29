# P3 — L'équipe qui adopte de l'extérieur

> **Verdict : 🔴 impossible en l'état.** Le projet n'a pas pu être écrit, et c'est le résultat.
>
> ✅ **Un verrou est tombé depuis** : la licence est passée en **Apache-2.0** le 2026-07-29 (ADR 020,
> #155), et c'est **cette preuve qui l'a provoqué**. P3 reste rouge — zéro paquet importable — mais
> son blocage n'est plus une décision, c'est du travail.

Mesuré le **2026-07-29**, sur `main`, dans la toolbox.

## Ce que P3 voulait faire

Écrire une liste de tâches qui **dépend** du socle — `require` dans son `go.mod`, `import` de ses
primitives — plutôt que de le copier. C'est la définition même de cette persona : *« découvre le
socle, l'évalue, et voudrait en **dépendre** plutôt que le copier »*.

## Ce qui s'est passé

### Tentative 1 — importer une primitive

Un projet tiers minimal, un `replace` vers le socle local, et un usage de `result.Result` :

```go
import "github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
```

```
main.go:6:2: use of internal package
    github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result not allowed
```

**Le compilateur Go refuse.** Ce n'est pas une convention qu'on pourrait contourner : `internal/` est
une règle du langage, appliquée par l'outil de compilation.

### Tentative 2 — importer ce qui n'est pas sous `internal/`

```
$ go list ./... | grep -v /internal/
github.com/SteelHeart/go-hexa-fp-starter/cmd/cli
github.com/SteelHeart/go-hexa-fp-starter/cmd/hexa
github.com/SteelHeart/go-hexa-fp-starter/cmd/server
github.com/SteelHeart/go-hexa-fp-starter/cmd/worker
github.com/SteelHeart/go-hexa-fp-starter/tools/covergate
```

Cinq paquets. Puis :

```
$ go list -f '{{.ImportPath}} → package {{.Name}}' ./... | grep -v /internal/
…/cmd/cli       → package main
…/cmd/hexa      → package main
…/cmd/server    → package main
…/cmd/worker    → package main
…/tools/covergate → package main
```

**Les cinq sont des `package main`.** Go interdit d'importer un paquet `main`. Le nombre de paquets
réellement importables par un tiers n'est donc pas cinq :

> **Il est ZÉRO.**

C'est plus fort que ce que la grille des personas annonçait — elle écrivait « 0 » en comptant les
binaires comme non importables, ce qui était juste, mais sans dire que **la totalité** de la surface
non-`internal` est constituée de `main`. Il n'existe aucun paquet à promouvoir : il faudra en
**créer** la frontière, pas la déclarer.

## Le tableau de critères de P3, remesuré

| Critère | Cible | Mesuré le 2026-07-29 | Verdict |
|---|---|---|---|
| Paquets importables depuis l'extérieur | > 0 | **0** — les 5 hors `internal/` sont des `package main` | 🔴 |
| Politique de versions et de dépréciation | écrite | **inexistante** — `git tag` rend **0 tag**, aucun `CHANGELOG` | 🔴 |
| Frontière API publique / interne | déclarée | **inexistante** — l'ADR 015 en fixe la méthode, pas le résultat | 🔴 |
| ~~Licence — existe~~ **licence permettant l'usage** | oui | **Apache-2.0** depuis le 2026-07-29, concession de brevet comprise (ADR 020, #155) | ✅ |
| Langue de l'API et du règlement | lisible par l'équipe | **le code est en anglais** depuis l'ADR 018 ; `rules/` reste en français | ⚠️ |

**Trois rouges, une réserve, un vert** — remesuré le 2026-07-29 après l'ADR 020. Le seul vert est la
licence, et il ne débloque aucun des trois rouges : il rend leur résolution **possible**, ce qu'elle
n'était pas.

## Ce que P3 a pu faire malgré tout

Rien qui la satisfasse. Elle peut **copier** le socle avec `hexa new` — c'est ce que fait P1, et cela
fonctionne. Mais copier est précisément ce que cette persona refuse : *« ce qu'elle tue :
`internal/` partout. Tant qu'il tient, ce dépôt **ne peut qu'être copié** — donc ce n'est pas encore
un framework, quelle que soit l'intention. »*

## Les deux réserves méritent d'être lues

**La licence.** Elle « existait », et le critère était donc formellement atteint. Mais elle disait
*tous droits réservés* : P3 n'avait le droit de rien faire, même une fois la frontière publiée. **Le
critère tel qu'il était écrit ne mesurait pas ce qui compte** — il fallait le reformuler en « licence
permettant l'usage », et il redevenait rouge.

> ✅ **C'est fait, et l'avertissement du bas de page a été suivi le jour même.** Le critère est
> reformulé, et la licence est passée en **Apache-2.0** (ADR 020, #155). Il est vert pour la bonne
> raison.
>
> **Cette preuve est la cause directe du changement.** Elle n'a pas seulement constaté un rouge :
> elle a montré que le rouge était **par décision**, donc insensible à tout travail — et que la
> décision contredisait la condition de transfert, qui exige **toutes** les personas éprouvées.
> C'est le seul relevé qui pouvait produire ce constat, parce qu'il a tenté de faire ce que P3 veut
> faire au lieu de le décrire.

**La langue.** L'ADR 018 a mis le code en anglais, ce qui lève l'obstacle sur le godoc. Le règlement
reste en français, et c'est une décision. Pour une équipe non francophone, `rules/` — ~70 règles —
reste illisible : c'est le coût d'entrée que P5 mesure de son côté.

## Ce que cette preuve apporte à l'ADR 015

L'ADR 015 dit que la frontière publique doit être **dérivée d'un usage mesuré**, et que la liste
d'imports d'une application réelle **est** la mesure.

Cette tentative fournit la première donnée : **l'import qu'un tiers écrit spontanément est
`internal/pkg/result`.** C'est la primitive du typage des erreurs, celle sans laquelle aucun cas
d'usage du socle ne se lit. Si une frontière doit commencer quelque part, elle commence là.

⚠️ Un point de donnée n'est pas une mesure. Il en faudra d'autres, et ils viendront des projets P1
et P2 — qui, eux, écriront du code métier réel.

## Ce qu'il faudrait pour que P3 passe au vert

Dans l'ordre, et aucun n'est un détail :

1. **Créer** des paquets publics — il n'y en a aucun à promouvoir, les cinq candidats sont des
   `main` ;
2. une **politique de versions** écrite, et un premier tag — `v0.1.0` attend #89 ;
3. ~~une **licence permettant l'usage**~~ — **fait** le 2026-07-29 : Apache-2.0 (ADR 020, #155).

~~Le troisième point rend les deux premiers sans objet à court terme : **P3 restera rouge tant que la
licence n'aura pas changé**, quel que soit l'état du code.~~

> ✅ **Le verrou est levé.** Cet avertissement a été écrit pour que personne n'entreprenne la
> frontière publique en croyant débloquer P3. Il a servi autrement : il a rendu visible que la
> décision de licence **contredisait la condition de transfert**, et c'est cette contradiction qui a
> été tranchée — en faveur de l'ouverture (ADR 020).
>
> 🔴 **Les points 1 et 2 restent entiers, et P3 reste rouge.** Ouvrir la licence ne crée aucun
> paquet importable ; le compte est toujours **zéro**. Ce qui change, c'est que le travail cesse
> d'être inutile.
>
> ⚠️ **Et il devient irréversible.** Sous Apache-2.0, tout paquet rendu importable est utilisable
> pour toujours — un paquet publié par excès ne se retire plus. L'ADR 015 exige que la frontière se
> **dérive d'un usage mesuré** ; cette exigence passe de prudente à indispensable. La première
> donnée est dans cette preuve : l'import qu'un tiers écrit spontanément est `internal/pkg/result`.
