# ADR 018 — La langue du code est l'anglais, celle du règlement le français

- **Statut** : acceptée
- **Date** : 2026-07-29
- **Issue** : [#34](https://github.com/SteelHeart/go-hexa-fp-starter/issues/34)
- **Remplace** : rien. **Tranche** une question ouverte depuis l'origine du dépôt.

## Contexte

Le dépôt est écrit dans **deux langues, mélangées à l'intérieur des mêmes fonctions**. Mesuré par
l'audit [#107](https://github.com/SteelHeart/go-hexa-fp-starter/issues/107), écart É-06 :

- **≈148 commentaires godoc en français contre ≈8 en anglais** dans `internal/` et `cmd/` ;
- des identifiants non exportés `abonner`, `composer`, `demarrer`, `depiler`, `envoyer`, `peupler`
  **dans les mêmes paquets** que `newServer`, `mustBroker`, `newDispatcher`, `discardLogger`.

Une recommandation existait — *anglais pour le godoc et les identifiants, français pour `rules/`* —
et n'était appliquée **nulle part**. Une recommandation qu'aucun outil ne fait respecter n'est pas
une règle : c'est une préférence, et le dépôt a une règle d'or contre exactement cela.

**La preuve que la dérive est invisible de l'intérieur** : l'ADR 016, acceptée le 2026-07-28,
affirme que les tests du générateur avaient *« des identifiants en français là où tout le reste du
dépôt est en anglais »*. C'était faux au moment où c'était écrit, dans un document qui fait foi, et
personne ne l'a relevé. Un ADR étant immuable, cette phrase reste — et cette ADR-ci est la seule
façon de la corriger sans l'effacer.

**Ce qui rend la question exigible maintenant** : le dépôt est **public** depuis le 2026-07-28. Le
godoc est la première chose qu'ouvre un évaluateur, et `pkg.go.dev` le rendra tel quel le jour où la
frontière publique existera (ADR 015).

## Décision

### La ligne de partage

| Artefact | Langue | Pourquoi |
|---|---|---|
| **Identifiants Go** — paquets, types, fonctions, variables, champs, constantes | **anglais** | Ils se mélangent aux identifiants de la bibliothèque standard et des dépendances, dans la même expression |
| **Commentaires et godoc** | **anglais** | Publiés avec le code, indissociables de lui |
| **Messages d'erreur internes** (`fmt.Errorf`, sentinelles `errors.New`) | **anglais** | Ils apparaissent dans les journaux et les traces, à côté des erreurs des dépendances |
| **Messages de journal** (`slog`) | **anglais** | Même raison, et ils sont agrégés avec ceux des bibliothèques |
| **Noms de fichiers de test** | **anglais** | Déjà le cas — vérifié par l'audit, zéro écart |
| **Contenu des tests** — commentaires, identifiants, messages `t.Errorf` | **anglais** | Un test est du code |
| `rules/`, `documentation/`, ADR | **français** | Ce n'est pas du code, ce n'est pas publié avec lui, et c'est la langue de travail |
| Messages de commit, titres et corps de PR, issues | **français** | Idem |
| **Clés de configuration** (`config/*.yaml`) et **noms d'événements** | **anglais** | Déjà le cas ; ce sont des identifiants |

### L'exception qui n'en est pas une : les messages destinés à l'utilisateur

**`domain.Error.Message` reste hors du champ de cette ADR.** Ce champ est documenté comme
*« destiné à l'utilisateur : aucun détail technique »*, il sort tel quel sur `422` :

```
l'adresse de courriel n'est pas valide
le mot de passe doit faire au moins 12 caractères
```

Ce n'est **pas** de la langue de code, c'est du **contenu produit**. Le traduire ici reviendrait à
changer le contrat de l'API sous couvert d'un lot de refactorisation, et — plus grave — à faire
croire que la question de l'internationalisation est réglée alors qu'elle ne l'est pas :
`internal/pkg/i18n` n'existe pas ([#12](https://github.com/SteelHeart/go-hexa-fp-starter/issues/12)).

**Une chaîne visible par un utilisateur final se choisit par une négociation de langue, jamais par
la langue dans laquelle le développeur a tapé.** Elle attend #12, et c'est écrit ici pour que
personne ne la traduise « au passage ».

### Les ADR et le règlement ne sont pas réécrits

L'ADR 001 à 017 restent en français, y compris leurs extraits de code. Un ADR est **immuable** : le
traduire effacerait la trace du moment où la décision a été prise, et dans quelle langue.

## Conséquences

### Assumées

- **La campagne est large** : ~35 000 lignes de Go, 169 fichiers de production, 334 fichiers de
  test. Elle se livre en **tranches par zone**, chacune avec sa PR et sa CI verte.
- La règle « PR mono-sujet ≤ 400 lignes » de `rules/workflow-git.md` **ne peut pas être tenue** sur
  une traduction. Dérogation explicite : une tranche de traduction est **mono-sujet par
  construction** et son diff est **mécaniquement vérifiable** — le comportement ne change pas, la
  CI le prouve. Découper plus fin produirait vingt PR dont aucune ne serait relisible isolément.
- **Le job CI `e2e` cherche des lignes de journal en français.** Il devra suivre la tranche qui
  traduit les journaux, dans la même PR — sinon il devient vert en ne trouvant plus ce qu'il
  cherchait, ce qui est précisément la forme de faux vert que l'ADR 013 combat.

### Refusées

| Option écartée | Pourquoi |
|---|---|
| **Tout français, assumé** | Cohérent et moins coûteux, mais le godoc est public et se lit à côté de la bibliothèque standard. Ferme la porte à toute contribution non francophone |
| **Rester à moitié** | Le pire des trois. Le mélange est *à l'intérieur des fonctions*, donc il n'existe aucune frontière à laquelle s'arrêter pour lire |
| **Traduire après le transfert** | Le godoc mixte serait public entre-temps, et la dette grossit à chaque module écrit |

## Outillage

**Une règle non outillée n'existe pas.** Un garde de langue est livré avec cette ADR — sinon la
recommandation de #34 se reproduirait à l'identique, un ADR de plus.

`tools/verifie-langue-du-code.sh` refuse, dans les lignes **ajoutées** d'une PR :

- un identifiant Go déclaré avec une **marque diacritique** (`é`, `è`, `à`, `ç`, `ô`…) ;
- un identifiant déclaré dont un segment figure dans une **liste close de mots français** observés
  dans le dépôt — `abonner`, `composer`, `demarrer`, `depiler`, `envoyer`, `peupler`, `boucler`,
  `lire`, `poser`, `verifier`, `refuser`… ;
- un **commentaire** contenant une marque diacritique, hors des chemins exclus.

**Ce garde ne prouve pas qu'un texte est anglais** — aucun garde bon marché ne le peut. Il attrape
le français **écrit comme du français l'est** : accentué, ou puisant dans un vocabulaire connu. La
limite est écrite ici plutôt que laissée à découvrir, et la liste de mots se complète à chaque faute
trouvée en revue.

Exclusions, nommées et motivées comme celles du garde de mention d'outillage, avec le même **garde
anti-pourriture** : une exclusion qui ne correspond plus à rien fait échouer la CI.

Il est livré avec `--temoin`, qui prouve les deux moitiés — il refuse le français **et** il sait
être satisfait par de l'anglais (ADR 013).
