# ADR 013 — Un garde est livré avec le cas qui le fait échouer

- **Statut** : Accepté
- **Date** : 2026-07-27
- **Remplace** : —
- **Issue** : #41

## Contexte

Le règlement de ce dépôt repose sur une règle d'or : **une règle non outillée n'existe pas**. Elle
est juste, et elle est incomplète. Elle dit qu'une règle doit avoir un outil ; elle ne dit rien de
l'outil lui-même.

Le 2026-07-27, une vérification systématique a donné le résultat suivant : **huit gardes écrits,
huit défectueux à leur premier allumage.** Aucun n'avait jamais été exécuté.

| Garde | Ce qu'il devait tenir | Premier verdict réel |
|---|---|---|
| Les 12 jobs de `ci.yml` | toute la barrière | **jamais démarrés** — 66 exécutions, 66 `startup_failure`, 0 succès depuis le premier commit (#47) |
| Crochets `commit-msg` et `pre-push` | message conforme, pas de push sur le tronc | **inertes** — versionnés en `100644`, donc ignorés par git sur toutes les machines (#43) |
| `gitleaks` | aucun secret versionné | 2 signalements au premier scan (#50) |
| Garde d'isolation des schémas | pas de SQL traversant | **9 faux positifs**, aucun vrai (#40) |
| Garde `inertia` | pas de mention d'outillage d'assistance | **se signale lui-même** : son motif est écrit en clair dans le fichier qui le définit |
| Garde `inertia`, second défaut | idem | signale `.githooks/commit-msg`, **le fichier qui porte la règle**. Bloquera la fusion vers `main` |
| `task rename` | isoler la seule valeur nominative | laissait le chemin dans `.golangci.yml` et `Dockerfile` (#48) |
| Niveau de test `e2e` | un parcours critique par surface | **0 test exécuté**, et `ok` affiché — sans `-tags=e2e`, zéro test compile |

Le point commun n'est pas la négligence. Dans **chaque** cas, le dispositif *paraissait* en place :
le fichier existait, le job était déclaré, la commande figurait dans le `Taskfile`. Ce qui manquait
était la seule chose qui distingue un garde d'un décor — **l'avoir vu refuser quelque chose**.

Un garde qui n'a jamais rougi n'a pas été vérifié. Il a seulement été écrit.

Cette faiblesse a une portée particulière depuis le glissement vers un **framework** : les gardes ne
protègent plus seulement ce dépôt, ils sont **livrés** aux projets engendrés. Un garde décoratif s'y
duplique à l'identique, et personne ne le découvre jamais — puisqu'il ne dit jamais non.

## Décision

**Tout garde est livré avec le cas qui le fait échouer.**

Modalités :

1. **Le cas d'échec est versionné et exécuté**, au même titre que le cas nominal. Un garde nouveau
   sans cas d'échec est incomplet, exactement comme un correctif sans test de non-régression.
2. **Le code de retour fait foi, jamais la sortie.** Une commande qui n'a pas tourné rend une sortie
   vide, ce qui ressemble à « propre ».
3. **Quand un compte nul est indiscernable d'un succès, le garde compte.** Le niveau `e2e` compte
   ses `=== RUN` et échoue à zéro ; le principe s'étend à tout garde dont l'absence de matière
   produit un vert.
4. **Un garde ne s'applique pas aux fichiers qui le définissent.** L'exclusion est **nommée et
   motivée** dans le code du garde — jamais un assouplissement du motif.
5. **Toute exception porte son motif écrit à côté**, plus la consigne de la retirer quand elle ne
   correspond plus à rien. Une liste d'exception non motivée grossit toujours, et le garde finit par
   ne plus rien garder.

## Conséquences

### Ce que ça achète

- La distinction entre « aucune violation » et « le garde ne tourne pas » devient **observable**.
  C'est la seule chose qui manquait aux huit cas ci-dessus, et elle les aurait tous attrapés.
- Un garde livré à un projet engendré arrive avec sa propre preuve de fonctionnement.
- Le coût de la vérification est payé **à l'écriture**, quand le contexte est frais, plutôt qu'au
  premier incident.

### Ce que ça coûte

- Chaque garde coûte environ le double à écrire : le refus doit être **provoqué**, ce qui demande
  souvent de fabriquer une situation invalide et de la maintenir.
- Certains cas d'échec sont désagréables à versionner — un faux secret, une migration volontairement
  fautive, un message de commit interdit. Ils doivent être **clairement identifiés comme témoins**,
  sans quoi le prochain lecteur les prendra pour des défauts.
- Un cas d'échec peut lui-même pourrir : le jour où le garde change, il faut le mettre à jour.

### Ce que ça rend impossible

- Livrer un garde « pour plus tard », en comptant sur le fait qu'il servira un jour.
- Conclure d'une CI verte qu'elle a vérifié quoi que ce soit, sans savoir ce qu'elle sait refuser.

## Alternatives écartées

| Alternative | Pourquoi écartée |
|---|---|
| Se contenter de la règle d'or existante (« une règle non outillée n'existe pas ») | Elle est vraie et insuffisante : les huit gardes ci-dessus étaient tous outillés. Ce qui manquait était la preuve que l'outil fonctionne |
| Revue humaine du garde | C'est précisément ce qui a échoué huit fois. Aucune relecture ne distingue un job qui va démarrer d'un job qui ne démarrera jamais |
| Tests de mutation généralisés (F003) | Bien plus coûteux, et ils répondent à une autre question — la qualité des assertions, pas l'exécution du garde. Complémentaire, pas substituable |
| N'appliquer la règle qu'aux nouveaux gardes | Laisse en place les huit défauts constatés, dont certains bloquent la fusion vers le tronc |

## Garde

Cette décision se vérifie elle-même, et c'est voulu : **le garde de cette règle est l'existence des
cas d'échec.**

- `task ci` rejoue les gardes et **rend un code de retour** ; sa documentation nomme explicitement
  ce qu'il ne couvre pas.
- Les cas d'échec déjà versionnés ou constatés : `gitleaks` sur un secret introduit, `ci:pr-title`
  sur un titre non conforme, `commit-msg` sur une mention interdite, `pre-push` vers le tronc,
  `rename:verify` sur une occurrence oubliée, `e2e` sur un compte de tests nul.
- **`[humain]`, et il faut le dire** : rien ne vérifie aujourd'hui qu'un garde *nouvellement ajouté*
  possède son cas d'échec. C'est une faiblesse connue de cet ADR, pas une tolérance. Elle se ferme
  le jour où la CI démarre (#47), par un inventaire exécuté.
