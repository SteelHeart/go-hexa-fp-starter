# ADR 017 — Le jeton authentifie, il n'autorise pas

- **Statut** : acceptée
- **Date** : 2026-07-28
- **Issue** : [#11](https://github.com/SteelHeart/go-hexa-fp-starter/issues/11)
- **Grave** trois arbitrages produit rendus le 2026-07-28, et **confirme** les trois principes déjà
  écrits dans [`documentation/technique/modules-noyau.md`](../technique/modules-noyau.md) § `auth`.

## Contexte

Le socle n'a **aucune** authentification. `documentation/produit/personas.md` en fait un critère
mesurable de **P1**, la persona primaire — « modules métier livrables sans authentification : cible
0, aujourd'hui tous » — et la lecture produit du dépôt est plus brutale encore : *un évaluateur pose
deux questions, obtient deux « non », et repart sans jamais découvrir que l'outbox est excellente.*

C'est aussi la ligne la plus lourde de la matrice `v0.1`, et la seule que le lead dev ne pouvait pas
commencer seul : trois questions dont les réponses ne se déduisent d'aucun fichier.

Elles ont été posées et tranchées. Cette ADR les grave, avec ce qu'elles coûtent.

## Décisions

### 1. Le jeton authentifie, il n'autorise pas

`Authorize` interroge **l'état persisté, à chaque appel**. Les permissions ne voyagent pas dans le
jeton.

Ce principe était déjà écrit ; il est gravé ici parce qu'il est **contre-intuitif et coûteux**, donc
exactement le genre de décision qu'on renégocie sans le vouloir. Mettre les permissions en
revendications du jeton est plus rapide, plus « moderne », et se défend en réunion : aucun aller-
retour, validation hors ligne.

Le coût de cette facilité est une **fenêtre de péremption**. Un droit retiré reste actif jusqu'à
l'expiration du jeton. Sur un jeton de quinze minutes, cela veut dire : *quinze minutes après avoir
révoqué un accès, il fonctionne encore*. Le jour où l'on révoque, c'est qu'on est pressé.

Conséquence assumée : chaque vérification touche le magasin d'autorisation. Le pilote par défaut
vivant en mémoire, cela reste gratuit en développement ; en production c'est une requête indexée,
et c'est le prix de la révocation immédiate.

### 2. Le cœur ne connaît qu'un jeton ; le transport dépend de la surface

Le cœur **émet et valide un jeton**, rien d'autre. Ce qui change d'une surface à l'autre, c'est
seulement la façon dont ce jeton voyage :

| Surface | Transport | Pourquoi |
|---|---|---|
| web | cookie `httpOnly`, `SameSite=Lax`, `Secure` hors développement | inaccessible au JavaScript, donc non volable par une seule faille XSS |
| mobile · CLI · service-à-service | en-tête `Authorization: Bearer` | pas de navigateur, donc pas de cookie ; et un en-tête se journalise moins facilement par accident |

C'est la propriété n°2 du socle — *le nombre de frontends est un non-sujet* — appliquée au sujet où
elle est le plus souvent trahie. Ajouter une surface n'ajoute pas un mode d'authentification : elle
choisit un **transport** pour le même jeton.

### 2 bis. Le jeton est OPAQUE, pas signé

Une chaîne aléatoire de 32 octets, retenue par le pilote. Pas de JWT, pas de clé
de signature, pas de rotation de clé.

Ce point a d'abord été écrit à l'envers dans cette ADR — « jeton signé » — par
réflexe. La décision 1 le rend pourtant sans objet : le seul avantage réel d'un
jeton signé est de **valider sans toucher au magasin**, or `Authorize` y va de
toute façon, à chaque appel. On paierait donc la gestion de clés pour un gain
déjà abandonné.

Ce qu'on évite, en plus, n'est pas mince : la famille de failles propre aux
jetons signés — algorithme `none` accepté, confusion HMAC/RSA, clé de
vérification devinée, expiration non vérifiée — n'existe pas sur une chaîne
aléatoire comparée à un enregistrement. Le meilleur code de sécurité reste celui
qu'on n'écrit pas.

Le prix est nommé : **chaque validation touche le magasin**. C'est le même prix
que la décision 1, pas un second.

### 3. La source d'identité est un PILOTE

`auth` suit l'anatomie de l'ADR 012 comme les six modules noyau existants :

- **`memory`** — magasin interne, aucune dépendance. C'est le défaut, et c'est ce qui rend vraie la
  promesse « `hexa new` puis `go run`, ça démarre » **y compris avec l'authentification**.
- **`postgres`** — le même magasin, durable.
- **`oidc`** — délégation à Keycloak, Zitadel, Auth0. **Déclaré dans la cible, PAS livré en v0.1**
  (voir « Périmètre » ci-dessous).

### 4. Le code vérifie une PERMISSION, jamais un rôle

Un rôle porte des permissions ; les appels de vérification portent sur la permission.

```go
// Ce que le code écrit
authorize(ctx, identity, "billing.invoice.void")

// Ce que le code n'écrit JAMAIS
if identity.Role == "admin" { … }
```

La différence ne se voit pas le premier jour et décide de tout au bout de trois ans. Avec des rôles
testés dans le code, ajouter un rôle est un **déploiement** ; avec des permissions, c'est une
**donnée**. Et la liste de ce que peut réellement un rôle redevient auditable — on la lit dans la
base, pas en parcourant les `if` du dépôt.

### 5. L'identité entre par la commande, jamais par le contexte

Un cas d'usage reçoit l'identité comme **champ de sa commande**. Il ne la lit pas d'un
`context.Value`.

Déjà écrit, gravé ici parce que c'est la première chose qu'on abrège quand on est pressé. Une
identité lue d'un contexte implicite rend le cas d'usage intestable sans construire un contexte, et
surtout elle rend **invisible** le fait qu'une décision métier dépend de qui appelle.

### 6. Le premier compte est AMORCÉ en développement, avec un secret engendré

La surface ne publie aucune opération d'administration : les exposer sans les protéger ouvrirait la
création de comptes à quiconque, et les protéger exige un premier administrateur. Un serveur neuf
rendait donc **401 à tout le monde**, sans exception — le délai avant premier succès du module était
*infini* (#99), très exactement le défaut que la tranche verticale avait supprimé pour
`user_registration`.

**Décision** : le composition root crée un compte `admin@local` **en développement et en test
uniquement**, avec un secret tiré de `crypto/rand` et affiché **une seule fois** au journal.

Trois propriétés rendent ce raccourci acceptable, et il faut les trois :

1. **Aucun secret par défaut n'existe dans un artefact versionné.** C'est la faute qui compte : un
   socle livré avec `admin/admin` est un socle qui *déploie* `admin/admin`, et personne ne le change
   avant l'incident.
2. **Hors `development` et `test`, rien n'est créé.** Pas une erreur — faire échouer le démarrage
   d'une production parce qu'elle refuse un compte de démonstration serait absurde — un **refus
   d'agir**, avec son test (ADR 013).
3. **L'opération est idempotente.** Redémarrer ne réinitialise aucun compte existant et ne rend
   aucun secret qu'on ne connaît pas.

**Ce que ça ne résout pas** : l'amorçage d'une *production*. Il reste à faire, et il n'a rien à voir
— il passera par une commande d'administration hors serveur (#8), pas par un raccourci de démarrage.
Écrire l'un ne dispense pas de l'autre, et les confondre est la manière dont un compte de
démonstration finit en production.

**Pourquoi le secret est journalisé plutôt que rendu par un autre canal** : parce que toutes les
alternatives sont pires ici. Un fichier serait versionné par accident ; une variable d'environnement
obligerait à en définir une pour qu'un `go run` fonctionne, ce qui rouvre le délai avant premier
succès. Le module, lui, ne journalise **pas** : il rend son compte rendu, et c'est le composition
root qui décide d'écrire. Cette frontière est ce qui empêche un secret de partir dans un collecteur
d'observabilité parce qu'un module a cru bien faire.

## Périmètre de la v0.1 — et ce qui est délibérément absent

| Livré en v0.1 | Déclaré, non livré |
|---|---|
| stratégie `password` (Argon2id, via `internal/infrastructure/security`) | `oauth2`, `oidc`, `saml_sso`, `apikey` |
| magasin `memory` (défaut) et `postgres` | fédération d'identités, liaison de comptes |
| RBAC → permissions | ABAC, ReBAC |
| jeton **opaque**, cookie web et porteur ailleurs | TOTP — il appartient à `notification`, pas à `auth` |

**Pourquoi ne pas écrire les autres pilotes tout de suite** : ce dépôt a déjà payé cette faute. Huit
paquets de pilotes ont vécu des mois avec **zéro test, à aucun niveau** (#37), et deux défauts de
production dormaient dedans — trouvés à la première exécution réelle. Un pilote `oidc` écrit sans
fournisseur pour l'éprouver serait du code qui *a l'air* de marcher. Il sera écrit quand un besoin
réel le testera.

⚠️ **`auth.enabled` vaut `false` dans `config/modules.yaml`**, et `true` dans
`config/env/development.yaml` — la même forme que la télémétrie. Deny par défaut en production ;
démontrable d'un `go run` en local.

## Conséquences

### Ce que ça achète

- **P1 cesse d'être bloquée sur son dernier critère technique.** Un module métier peut exiger une
  permission sans écrire d'infrastructure.
- Les deux « non » que reçoit tout évaluateur produit deviennent un « non » — `tenancy` (#23) reste.
- La révocation est **immédiate**, sans fenêtre de péremption à expliquer après un incident.

### Ce que ça coûte

- **Une requête d'autorisation par vérification.** Mesurable, indexée, et c'est le prix explicite de
  la décision 1. Le jour où elle deviendra le goulot, la réponse sera un cache décoré sur le port de
  lecture — pas des permissions dans le jeton.
- **`internal/core/auth` retourne `error`, pas `Result[T, domain.Error]`.** L'invariant du dépôt
  tient — un module noyau est technique — mais `auth` a bel et bien une taxonomie que les surfaces
  doivent traduire en 401, 403 et 422. Elle passe donc par des **erreurs sentinelles** énumérées,
  reconnaissables par `errors.Is`. C'est moins expressif qu'un `Result`, et c'est le prix de
  l'homogénéité du noyau.
- **Deux magasins à garder cohérents** : les identités et les permissions. Le pilote `postgres`
  devra les tenir dans la même transaction, sous le schéma `platform` (ADR 011).

### Ce que ça rend impossible

- Qu'un droit révoqué continue de fonctionner — la décision 1 l'exclut par construction.
- Qu'une surface invente son propre mode d'authentification : le cœur ne connaît qu'un jeton.
- Qu'un rôle soit testé dans le code : le port ne prend qu'une permission.

## Alternatives écartées

| Alternative | Pourquoi non |
|---|---|
| **Permissions dans les revendications du jeton** | Plus rapide, et crée une fenêtre pendant laquelle un accès révoqué fonctionne encore. Le jour où l'on révoque, on est pressé |
| **Jeton porteur pour toutes les surfaces, cookie compris** | Une seule mécanique, plus simple à écrire — et le jeton devient lisible par du JavaScript sur le web, donc volable par une seule faille XSS |
| **Jeton signé (JWT)** | Valide sans toucher au magasin — un gain que la décision 1 a déjà abandonné, puisque `Authorize` y va à chaque appel. On paierait la gestion de clés, leur rotation, et toute la famille de failles propre aux jetons signés, pour rien |
| **Déléguer entièrement à un fournisseur OIDC** | Sort du périmètre le sujet le plus risqué du produit — et supprime tout démarrage sans infrastructure, plus la raison d'être de `user_registration` |
| **ReBAC façon Zanzibar** | Ce qu'il faut pour du partage fin de documents. Un service de relations à part entière, hors de proportion pour une v0.1 |
| **Compte d'amorçage à mot de passe fixe dans la configuration** | Le plus simple, et le seul vraiment dangereux : un défaut versionné est un défaut déployé. C'est `admin/admin`, et il survit à toutes les revues |
| **Amorçage par variable d'environnement obligatoire** | Aucun secret journalisé — mais un `go run` sur une machine vierge échoue tant qu'on n'a pas deviné quelle variable définir, ce qui rouvre le délai avant premier succès que la décision 6 vient de fermer |
| **Exposer l'inscription publiquement** | Un serveur de démonstration utilisable en trente secondes, et une création de comptes ouverte à tout le réseau. C'est la faute que la décision 6 contourne, pas celle qu'elle accepte |

## Garde

- **`arch-go`** : `internal/core/**` ne dépend d'aucun module métier. `auth` ne peut donc pas
  connaître `user_registration` — leur liaison appartient au composition root, et le jour venu au
  consommateur d'événements (#9).
- **Un test doit constater qu'une permission RÉVOQUÉE est refusée à l'appel suivant**, sans
  expiration de jeton. C'est le témoin de la décision 1, et il échouerait si quelqu'un déplaçait les
  permissions dans le jeton (ADR 013).
- **Un test doit constater qu'aucune vérification ne porte sur un rôle** : le port n'accepte qu'une
  permission, donc le compilateur s'en charge.
- **Un test doit constater que l'amorçage ne crée RIEN hors développement**, et il vérifie les deux
  moitiés : le compte rendu est vide *et* aucune identité n'existe réellement. Un compte rendu vide
  se falsifie en une ligne ; un magasin peuplé, non. C'est le témoin de la décision 6.
- **[humain]** Rien n'empêche mécaniquement une surface de lire l'identité d'un `context.Value`. La
  décision 5 est une règle de revue tant qu'aucun garde ne sait la vérifier.
