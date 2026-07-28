# ADR 014 — Le catalogue des modules est une valeur passée au chargeur, pas une table du framework

- **Statut** : Accepté
- **Date** : 2026-07-27
- **Remplace** : —
- **Issue** : [#76](https://github.com/SteelHeart/go-hexa-fp-starter/issues/76)

## Contexte

`internal/config/modules.go` porte deux tables au niveau paquet — `knownDrivers` et
`defaultDrivers` — qui énumèrent les six modules noyau et leurs pilotes. Un module absent de
`knownDrivers` est refusé à la validation.

C'est un excellent deny-par-défaut. Pour le socle.

### La friction, mesurée

Ajouter un module métier à la configuration **livrée**, puis démarrer :

```yaml
  billing:
    enabled: true
    driver: postgres
```

```
démarrage impossible: configuration: configuration invalide:
  modules.billing : module inconnu (voir documentation/technique/pilotes.md)
```

Code de retour **1**. Ce n'est pas une déduction depuis le code : c'est la réponse du binaire.

Le message renvoie même vers `documentation/technique/pilotes.md` — un document **du framework**.
On envoie donc l'auteur de `billing` consulter un catalogue qui ne lui appartient pas, pour y
chercher un module qu'il vient d'écrire.

### Ce que ça coûte réellement

Trois capacités que le socle s'offre et refuse à l'application : `enabled`, `driver`, `options`.

Aujourd'hui `user_registration` s'en tire parce qu'il est livré **avec** le socle : son `module.go`
valide son propre pilote, et `cmd/server` lit `cfg.Modules[Name].Driver` — un champ qui **reste
vide**, puisque la configuration refuserait de le porter. La tranche de référence contourne donc
silencieusement le mécanisme qu'elle est censée démontrer.

Ça ne tiendra pas pour une application. Faire modifier `internal/config/modules.go` — un fichier du
framework — pour déclarer le pilote de son propre module `billing` est **exactement** la friction
qu'un framework ne doit pas avoir. C'est l'un des trois blocages relevés pour la persona primaire
dans [`documentation/produit/personas.md`](../produit/personas.md) : *créer, brancher et configurer
un module exigent tous de modifier le framework.*

### Ce qui contraint la solution

| Contrainte | Effet sur la conception |
|---|---|
| **`arch-go` règle 7** — `internal/config` ne dépend d'**aucun** paquet interne | Le catalogue ne peut pas y **nommer** les modules. Il doit être **reçu**, pas écrit |
| **`arch-go` règle 4c** — `internal/core/**` ne connaît aucun module métier | Le noyau ne peut pas agréger le catalogue de l'application |
| **ADR 004** — composition manuelle, aucun conteneur | Ni `init()`, ni registre global mutable |
| Les modules noyau importent **déjà** `internal/config` (`outbox.New` reçoit un `config.Module`) | La direction de dépendance module → config existe et reste licite |

Ces quatre contraintes ne laissent qu'une place possible au catalogue : **entre les mains du
composition root**, seul code autorisé à tout connaître.

### Ce qu'on ne sait pas encore

Le catalogue décrira-t-il un jour davantage qu'un ensemble de pilotes — des besoins en ressources,
une compatibilité de version, un ordre de montage ? **On l'ignore.** La décision porte sur le
mécanisme de transmission, pas sur la richesse de ce qui transite. Un champ s'ajoute à une structure
sans rien casser ; une table globale, elle, ne se déplace jamais sans tout réécrire.

## Décision

**Le catalogue des modules est une valeur, construite par le composition root et passée au chargeur
de configuration. Chaque module — noyau comme métier — déclare ses pilotes dans son propre
`module.go`.**

Modalités :

1. **`internal/config` définit les types, jamais le contenu.** Un `ModuleCatalog` associe un nom de
   module à l'ensemble de ses pilotes admis et à son pilote par défaut. Le paquet ne cite plus aucun
   nom de module — ce qui le remet en accord avec la règle 7, qu'il respectait à la lettre et
   contournait dans l'esprit.

2. **La validation prend le catalogue en paramètre.** `Load` le reçoit ; sans lui, la configuration
   ne peut rien admettre. Un catalogue vide refuse **tout** : le deny-par-défaut ne se relâche pas,
   il change seulement de source.

3. **Chaque module expose son catalogue depuis son `module.go`**, dans un fichier dédié —
   `catalog.go`, un fichier par fonction publique. Il vit donc à **côté** de la fabrique qui
   construit les pilotes, dans le même paquet. C'est le point qui compte : la table de validation et
   le `switch` de construction ne peuvent plus diverger, alors qu'ils vivent aujourd'hui dans deux
   paquets différents et que le commentaire de `knownDrivers` avoue déjà craindre cette dérive.

4. **Le composition root fusionne.** `cmd/server` et `cmd/worker` assemblent le catalogue du noyau et
   celui des modules métier, puis le passent au chargeur. Un module qui n'est pas monté par le
   composition root n'est pas dans le catalogue, donc n'est pas configurable : **on ne configure pas
   ce qu'on n'a pas branché.**

5. **Aucun enregistrement implicite.** Pas d'`init()`, pas de registre mutable, pas d'effet de bord à
   l'import. Un module devient connu parce que quelqu'un l'a **écrit** dans la composition, et cette
   ligne se lit.

## Conséquences

### Ce que ça achète

- **Un module métier se déclare sans toucher un fichier du framework.** C'est la friction nommée,
  supprimée à sa racine — pas contournée.
- **La validation et la fabrique cohabitent.** Une faute de frappe entre les deux devient
  impossible : c'est le même paquet, souvent le même écran.
- **La frontière framework / application devient lisible dans les types.** Ce que le socle fournit,
  ce que l'application ajoute, et le point exact où les deux se rejoignent.
- **`hexa new` (#17) devient concevable.** Un module généré apporte son propre catalogue ; le
  générateur n'a aucun fichier du framework à réécrire. Sans cette décision, il en aurait eu un — et
  un générateur qui modifie le framework qu'il instancie n'est pas un générateur.
- **Le monorepo (#16) cesse d'être bloqué par ce point.** `core/` peut partir sans emporter la liste
  des modules de l'application.

### Ce que ça coûte

- **Le composition root grossit.** Deux lignes de plus par module. C'est le prix assumé de l'ADR 004,
  et c'est le bon endroit pour le payer : ce code est fait pour tout connaître.
- **Une étape de plus pour ajouter un module noyau** au socle lui-même : déclarer son catalogue.
  Aujourd'hui il suffit d'ajouter une ligne à une table. Le gain n'est pas symétrique — le socle perd
  un peu de confort pour que l'application en gagne beaucoup.
- **Une rupture d'API.** `config.Load` change de signature. Le dépôt n'a pas encore publié de
  `v0.1.0` : c'est **maintenant** ou jamais, et c'est précisément pourquoi cette décision passe avant
  le tag et non après.
- **`RequiresSQL` et `RequiresCache` ne peuvent plus itérer une table globale.** Elles doivent
  recevoir le catalogue, ou en dépendre par la valeur qui les porte. Un point de conception à
  trancher à l'implémentation, pas ici.

### Ce que ça rend impossible

- **Configurer un module qui n'est pas monté.** Le catalogue vient du composition root : ce qui n'y
  figure pas n'existe pas pour la configuration. C'est voulu — un module « configuré mais non
  branché » se comporterait comme un module désactivé, en silence.
- **Découvrir les modules par réflexion ou par import.** Aucune liste ne se peuple toute seule. Le
  jour où quelqu'un voudra un plugin chargé dynamiquement, cette décision sera dans son chemin, et il
  faudra un ADR pour la remplacer.

## Alternatives écartées

| Alternative | Pourquoi écartée |
|---|---|
| **Statu quo** — le module métier valide son pilote dans son seul `module.go` | Mesuré : la configuration **refuse** de porter son nom. `enabled`, `driver` et `options` lui restent inaccessibles, et `cmd/server` lit un champ toujours vide. Le socle s'offre trois capacités qu'il refuse à l'application |
| **Enregistrement par `init()`** dans un registre global | Contredit l'ADR 004. Un module deviendrait connu **parce qu'il est importé**, pas parce qu'on l'a branché : l'inverse exact du composition root. Dépend de l'ordre d'initialisation, invisible à la lecture, et intestable sans effets croisés entre tests |
| **Ouvrir le schéma** : accepter n'importe quel nom de module, valider seulement dans `module.go` | Détruit le deny-par-défaut au niveau configuration. `bilingue:` au lieu de `billing:` serait **accepté**, et le module resterait silencieusement désactivé. C'est le mode de défaillance que ce dépôt combat le plus — celui qui ne se signale jamais |
| **Une section `custom:` séparée** dans `modules.yaml` pour les modules applicatifs | Deux vocabulaires pour un seul concept. Un module métier n'est pas un citoyen de seconde zone : l'ADR 012 pose **une seule anatomie, deux provenances**. Une section à part le démentirait dans le fichier que tout le monde lit en premier |
| **Un fichier de catalogue déclaratif** (`catalog.yaml`) lu au démarrage | Déplace le problème sans le résoudre : il faudrait alors garder l'accord entre ce fichier et le code des fabriques, c'est-à-dire recréer exactement la divergence que la décision 3 supprime |

## Garde

- **`arch-go`** vérifie mécaniquement que `internal/config` ne dépend d'aucun paquet interne
  (règle 7) et que `internal/core/**` ignore `internal/modules/**` (règle 4c). Ce sont ces deux
  règles qui rendent l'alternative « table dans le framework » **impossible à réintroduire sans
  faire rougir `task check`** : la décision est tenue par l'outil, pas par la vigilance.
- **Un test de configuration** doit prouver qu'un module absent du catalogue **refuse** le
  démarrage — le deny-par-défaut vérifié, pas supposé.
- **Un test de bout en bout** doit prouver qu'un module métier fictif, déclaré uniquement par son
  propre catalogue, est accepté par la configuration **sans qu'aucun fichier du framework ne le
  nomme**. C'est le sens de l'ADR 013 appliqué ici : la décision est livrée avec le cas qui la
  démontre, et ce cas échouerait si quelqu'un remettait une table globale.
- **[humain]** Rien ne vérifie mécaniquement que le catalogue d'un module reste en accord avec sa
  fabrique. La décision 3 rend l'écart **improbable** en les mettant dans le même paquet ; elle ne
  le rend pas impossible. C'est un aveu de faiblesse, pas une tolérance — le jour où un garde saura
  comparer les deux, il devra être écrit.

## Extension du 2026-07-28 — les OPTIONS suivent le même chemin (#93)

Cette ADR déclarait les **pilotes** d'un module hors du framework. Elle laissait leurs **options**
sans schéma, et le trou était mesurable :

```yaml
outbox:
  driver: memory
  options:
    bath_size: 50      # au lieu de batch_size
```

Le serveur **démarrait**, montait le module, et n'en disait rien. Les accesseurs — `IntOption`,
`DurationOption`… — rendent la valeur par défaut quand la clé est absente, ce qui est correct pris
isolément ; mais rien n'énumérait les clés connues, donc *absente* et *mal orthographiée* étaient
indiscernables. Le deny-par-défaut s'arrêtait au nom du pilote.

`config.Resources` porte désormais `Options []string`, déclarées par le module **à côté du code qui
les lit**, en partageant ses constantes — exactement le dispositif de la décision 3. Une clé
inconnue refuse le démarrage en nommant les clés admises.

Trois points valent d'être notés :

- **La déclaration est PAR PILOTE, pas par module.** `flags` et `settings` n'ont de sens que pour le
  pilote `file` de `dynconf` ; `namespace` que pour le pilote `redis` d'`idempotency`. Une liste par
  module accepterait une option sans effet — le réglage qu'on ne découvre jamais.
- **La valeur zéro n'admet rien.** Un pilote qui oublie de déclarer ses options voit sa
  configuration refusée, ce qui se remarque. L'inverse rouvrirait le trou pour tous les pilotes.
- **La faiblesse [humain] ci-dessus se réduit sans disparaître.** Le partage de constante empêche
  qu'une clé lue diffère d'une clé admise ; il n'empêche pas d'admettre une clé que plus personne ne
  lit.
