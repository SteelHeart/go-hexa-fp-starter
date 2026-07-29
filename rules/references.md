# Où trouver quoi

## Dans ce dépôt

| Je cherche | C'est ici |
|---|---|
| Le règlement d'ingénierie | [`rules/`](README.md) — **fait foi** |
| Ce qui est absolument interdit | [`rules/interdictions.md`](interdictions.md) |
| La barre pour livrer | [`rules/definition-of-done.md`](definition-of-done.md) |
| Une décision d'architecture et son pourquoi | [`documentation/adr/`](../documentation/adr/README.md) — **fait foi** |
| Comment nommer une branche, un commit, un fichier | [`documentation/process/NOMENCLATURE.md`](../documentation/process/NOMENCLATURE.md) |
| Les labels et le board | [`documentation/process/LABELS.md`](../documentation/process/LABELS.md) |
| Les failles ouvertes | [`documentation/securite/registre-securite.md`](../documentation/securite/registre-securite.md) |
| Qui a le droit d'appeler quoi | [`documentation/securite/matrice-acces.md`](../documentation/securite/matrice-acces.md) |
| Ce qui manque au socle et fait perdre du temps | [`documentation/process/JOURNAL_FRICTION.md`](../documentation/process/JOURNAL_FRICTION.md) |
| Un exemple de module métier | `internal/modules/user_registration/` |
| Le contrat d'API | `api/openapi.yaml` — **généré**, jamais édité à la main |

## Hiérarchie des sources en cas de contradiction

1. **ADR** — tranche l'architecture. Gagne sur tout le reste.
2. **`rules/`** — règlement d'ingénierie.
3. **`documentation/AMORCAGE.md`** — amorçage, résumé. Ne fait jamais foi contre un ADR ou une règle.
4. **README, commentaires de code** — indicatifs.

Une contradiction découverte se corrige **dans la même PR** que celle qui l'a révélée. Une
contradiction laissée en place rend tout le corpus non fiable, et donc inutilisé.

## Le socle est réutilisable tel quel

Ce dépôt ne dépend d'aucune personne ni d'aucune organisation :

- aucun pseudo, aucune équipe, aucun `CODEOWNERS` — les contraintes portent sur des **règles**,
  vérifiées par la CI ;
- le **chemin de module** est la seule valeur nominative, isolée derrière `task rename` ;
- le module métier `user_registration` est un **exemple de référence**, à supprimer ou remplacer — il
  n'est requise par rien d'autre que ses propres tests.

Pour démarrer un projet à partir de ce socle, voir [`toolchain.md`](toolchain.md) § 7.

## Vocabulaire imposé

Pour que les recherches dans le code et la documentation aboutissent, un concept porte **un seul**
nom :

| On écrit | On n'écrit pas |
|---|---|
| **module** | composant, brique, paquet |
| **module noyau** | module framework, module systeme |
| **module metier** | feature, bounded context, domaine |
| **pilote** | driver, backend, implementation, provider |
| **surface** | canal, frontend, delivery |
| ~~**service**~~ | **PROSCRIT** \u2014 signifie deja microservice, couche service, unite systeme |
| **port** | interface, contrat, repository, gateway |
| **adaptateur primaire** | contrôleur, handler, delivery, entrypoint |
| **adaptateur secondaire** | repository, infrastructure, DAO |
| **cas d'usage** | service, interactor, business logic |
| **décorateur** | middleware (réservé au HTTP), wrapper, intercepteur |
| **composition root** | container, DI, bootstrap, wiring |

## Langue — ADR 018

**Le CODE est en anglais. Le RÈGLEMENT est en français.** La frontière passe entre ce qui est
publié avec le code et ce qui ne l'est pas.

| Artefact | Langue |
|---|---|
| Identifiants Go — paquets, types, fonctions, variables, champs, constantes | **anglais** |
| Commentaires et godoc | **anglais** |
| Messages d'erreur internes (`fmt.Errorf`, sentinelles) et messages de journal | **anglais** |
| Contenu des tests — noms de fichiers, identifiants, commentaires, `t.Errorf` | **anglais** |
| Clés de configuration, noms d'événements | **anglais** |
| `rules/`, `documentation/`, ADR | **français** |
| Messages de commit, titres et corps de PR, issues | **français** |

### La seule exception, et elle n'en est pas une

**`domain.Error.Message` est hors champ.** Ce champ sort tel quel sur `422`, donc c'est du
**contenu produit**, pas de la langue de code. Le traduire changerait le contrat de l'API sous
couvert de refactorisation, et ferait croire que l'internationalisation est réglée alors qu'elle ne
l'est pas — `internal/pkg/i18n` n'existe pas (issue #12).

Une chaîne visible par un utilisateur final se choisit par une **négociation de langue**, jamais par
la langue dans laquelle le développeur a tapé.

### Le garde, et ce qu'il ne sait pas faire

`tools/verifie-langue-du-code.sh` refuse, **dans les lignes ajoutées** d'une PR : un identifiant Go
accentué, un identifiant puisant dans une liste close de mots français, un commentaire accentué.

**Il ne prouve pas qu'un texte est anglais** — aucun garde bon marché ne le peut. Il attrape le
français écrit comme le français l'est. La liste de mots **se complète à chaque faute trouvée en
revue** : c'est le mécanisme prévu, pas un aveu.

Il ne regarde que les lignes ajoutées parce que la traduction se livre en **tranches** : un garde
rouge jusqu'à la dernière serait ignoré, donc désarmé. Il empêche la dette de croître pendant qu'on
la résorbe.
