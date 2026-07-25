# ADR 009 — Stratégie d'accès aux données : pile en couches, pas d'ORM unique

- **Statut** : Accepté
- **Date** : 2026-07-25
- **Remplace** : la formulation « zéro ORM » de la version initiale de `rules/donnees-et-migrations.md`

## Contexte

La question posée était : existe-t-il un ORM assez solide et flexible pour couvrir tous les
besoins, y compris le RLS Postgres, tout en laissant écrire du SQL pur ?

Après examen, **non** — et l'interdiction absolue précédente était tout aussi fausse dans l'autre
sens. Le paysage Go se répartit en trois familles aux compromis distincts :

| Outil | Nature | RLS | SQL pur | Fuite dans le domaine |
|---|---|---|---|---|
| **sqlc** | Générateur SQL → Go typé ; **le SQL est la source de vérité** | natif | par construction | aucune |
| **go-jet** | Constructeur de requêtes typé, généré depuis le schéma | oui | oui | aucune |
| **squirrel** | Constructeur de requêtes non typé | oui | oui | aucune |
| **Bun** | ORM léger, très proche de SQL | oui | trivial | tags de struct |
| **Ent** | ORM à génération de code, couche *Privacy* applicative | partiel — *Privacy* n'est pas le RLS Postgres | échappatoires | son schéma devient source de vérité, concurrent des migrations |
| **GORM** | ORM à réflexion | possible | oui | hooks, `gorm.Model`, *soft delete* — fuit partout |

Aucun ne couvre seul tous les besoins : **sqlc est excellent en statique et faible en dynamique**
(filtres composables, tri et pagination variables) ; les constructeurs de requêtes sont l'inverse.

Sur le **RLS** précisément, ce qui compte n'est pas l'ORM mais le **contrôle de l'acquisition de
connexion** : il faut garantir que chaque requête tourne dans une transaction où
`SET LOCAL app.current_tenant` a été posé, et qu'aucune connexion ne revienne au pool avec un état
résiduel. C'est ce que donne l'accès direct au pilote — et ce que tout ORM rend plus difficile à
*garantir*, même quand il le rend possible.

## Décision

Une **pile en couches**, pas un ORM unique :

| Couche | Outil | Part attendue |
|---|---|---|
| Requêtes statiques | **sqlc** (SQL écrit à la main, types générés) | ~90 % |
| Requêtes dynamiques | **squirrel** (filtres, tri, pagination composables) | ~9 % |
| Cas exotiques | **pgx** nu : `COPY`, `LISTEN/NOTIFY`, variables de session RLS | ~1 % |

Règles d'usage :

- **GORM est interdit partout.** Il ne se confine pas : ses hooks et son modèle embarqué
  contaminent les entités par conception.
- **Ent, Bun et squirrel sont autorisés, mais confinés** à `adapters/secondary/**` et
  `infrastructure/database/**`. `depguard` refuse leur import ailleurs.
- Les types générés par sqlc **ne traversent jamais** la frontière de l'adaptateur : ils sont
  traduits en *value objects* du domaine, dans le même fichier qui les produit.
- Le RLS est porté par le socle : `database.RunInTx` accepte une portée de tenant et pose
  `SET LOCAL` avant toute requête de la transaction.

## Conséquences

### Ce que ça achète

- Le SQL reste lisible, revisable et optimisable — c'est lui qu'on lit en incident.
- Le typage vient de la base réelle (sqlc analyse le schéma), donc un `ALTER TABLE` casse la
  compilation plutôt qu'un scan à l'exécution.
- Le RLS est garanti par construction : il n'existe pas de chemin d'accès hors transaction scopée.
- La porte reste ouverte : un besoin qui justifie Ent peut l'introduire sans toucher au cœur.

### Ce que ça coûte

- **Trois outils plutôt qu'un** : une courbe d'apprentissage répartie, et une frontière à tenir
  (quand passer de sqlc à squirrel).
- sqlc ajoute une étape de génération, donc un artefact généré à versionner et à régénérer.
- Écrire le SQL à la main coûte plus cher qu'un `db.Where(...).Find(...)` sur du CRUD simple.
  C'est assumé : le socle optimise le cas où la requête compte.

### Ce que ça rend impossible

- Faire remonter un type de persistance jusqu'au domaine.
- Accéder à une table policée hors d'une transaction scopée.

## Alternatives écartées

| Alternative | Pourquoi écartée |
|---|---|
| **GORM** | Fuit par conception ; incompatible avec un domaine pur |
| **Ent** seul | Son schéma devient une seconde source de vérité, concurrente des migrations ; sa couche *Privacy* est applicative et ne remplace pas le RLS |
| **Bun** seul | Bon compromis, mais ses tags de struct poussent à réutiliser les modèles de persistance comme entités |
| **pgx nu seul** (décision initiale) | Correct mais coûteux : scan à la main, aucune vérification du SQL à la compilation, requêtes dynamiques pénibles |
| **go-jet** à la place de sqlc | Équivalent en qualité ; sqlc retenu parce que le SQL reste du SQL lisible plutôt qu'une expression Go |

## Garde

`depguard` (règles `no-gorm` et `orm-confine`), `arch-go` (le cœur n'importe aucun paquet de
persistance), et la revue pour la traduction des types générés en *value objects*.
