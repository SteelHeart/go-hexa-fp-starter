# Journal de friction

Ce que le socle ne fournit pas et qui fait perdre du temps, ou pire, qui pousse à contourner une
règle. Une friction non écrite se paie à chaque fois qu'elle se produit, sans jamais devenir
visible.

**Format** : `F{NNN}` · date · ce qui bloque · ce que ça coûte · statut.
Une friction ouverte a son issue, avec le label `friction`.

| Réf | Date | Friction | Coût | Statut |
|---|---|---|---|---|
| F001 | 2026-07-25 | Docker absent de la machine de développement de référence | Les tests d'intégration, de bout en bout et les migrations ne sont pas exécutables en local. La CI est le seul endroit où ils tournent, donc la boucle de retour sur le SQL est de plusieurs minutes | **RÉSOLU** le 2026-07-27 — poste WSL sous **Podman rootless**, plus [`deploy/toolbox/`](../../deploy/toolbox/README.md) qui porte l'outillage en image. Le poste ne reçoit qu'un script `sh`. La pile démarre, `psql` et `goose` sont disponibles, et la boucle de retour sur le SQL passe de plusieurs minutes à quelques secondes. **Ce que cette ouverture a révélé** : la chaîne de migration ne fonctionnait nulle part, CI comprise — voir la section ci-dessous |
| F002 | 2026-07-25 | La protection de branche serveur exige un plan payant sur dépôt privé | Le crochet `pre-push` est un filet contournable avec `--no-verify`, pas un contrôle | **Ouvert** — arbitrage issue #18 : dépôt public, GitHub Pro, ou assumer l'absence |
| F003 | 2026-07-25 | Aucun test de mutation | La couverture mesure ce qui est **exécuté**, pas ce qui est **vérifié** : 90 % sur le cœur ne prouve pas que les assertions sont justes | **Ouvert** |
| F004 | 2026-07-25 | Les versions d'outillage sont en `latest` en CI | Un linter qui change de comportement rend la CI non reproductible et peut casser une PR sans rapport | **Ouvert** — à figer à la première release ([`rules/dependances.md`](../../rules/dependances.md) §5) |
| F005 | 2026-07-25 | `-race` exige CGO et un compilateur C, absents de la machine de référence Windows | Le détecteur de courses ne tourne qu'en CI. Un défaut de concurrence introduit en local n'est vu qu'à la PR | **RÉSOLU** le 2026-07-27 — `gcc` et `musl-dev` sont dans la toolbox. `task test:race` vert en local, code de retour 0. La correction vit dans une image, donc elle vaut pour tout poste qui la construit |
| F006 | 2026-07-25 | `task` (go-task) et `govulncheck` n'étaient pas installés sur la machine de référence | `task check` n'avait **jamais** été exécuté tel quel : les étapes tournaient une par une, donc rien ne garantissait que l'enchaînement était le même qu'en CI | **RÉSOLU** le 2026-07-26 — `go install` des deux (ce sont des outils Go, donc multiplateformes). `task check` a enfin tourné : `fmt` `vet` `lint` `arch` `vuln` verts, `test` bloqué par **F008** |
| F007 | 2026-07-25 | Chaîne d'outils Go en 1.25.4, **20 vulnérabilités de la bibliothèque standard** atteignables | `govulncheck` échouait, donc `task check` ne pouvait pas être vert | **RÉSOLU** le 2026-07-26 — `go 1.25.12` dans `go.mod` ; `GOTOOLCHAIN=auto` télécharge la chaîne. **Aucune installation système**, la correction vit dans le dépôt et vaut pour tout le monde. `govulncheck` : 0 vulnérabilité |
| F009 | 2026-07-26 | Le cliquet de couverture mesurait **3,6 %** au lieu de **52,4 %** | Les tests sont en boîte noire, dans `{paquet}/tests/` : sans `-coverpkg=./...`, le profil n'attribue la couverture qu'au paquet de test. Le cliquet aurait échoué pour une raison fausse, et la tentation aurait été de baisser le seuil. Jamais vu parce que le job `test` de la CI n'avait pas encore tourné | **Partiellement résolu** — `-coverpkg=./...` ajouté au `Taskfile` et à la CI. **Reste que 52,4 % < 70 %** : `messaging`, `modulebus`, `httpserver`, `telemetry`, `cache` ont zéro test. Ne pas baisser le seuil |
| F008 | 2026-07-26 | **Un programme Go ne peut créer AUCUN fichier sous `C:\xampp\htdocs\`** — le shell, lui, y écrit sans problème. **Cause identifiée** : accès contrôlé aux dossiers de Windows Defender, `C:\xampp\htdocs\dev` étant dans la liste protégée | `task check` ne peut pas être vert : `go test -coverprofile=coverage.out` échoue sur « Le fichier spécifié est introuvable ». Même cause pour `go get` qui ne peut pas réécrire `go.mod`. Diagnostiqué par un programme témoin de dix lignes : création refusée dans le dépôt, acceptée dans `%TEMP%` et dans `C:\Users\MAC\`. C'est une protection du répertoire web de XAMPP — antivirus ou accès contrôlé aux dossiers | **Ouvert** — **`task check` a été vérifié VERT (exit 0) depuis un clone hors zone protégée**, les six étapes. Ce n'est donc pas un défaut du dépôt. Remèdes : sortir le dépôt de `htdocs`, travailler sous **WSL** (prévu), ou autoriser `go.exe` dans Defender — ce dernier point relève de l'utilisateur. **Ne pas désactiver la protection** : c'est une garde anti-rançongiciel délibérée |

| F010 | 2026-07-27 | Le garde CI « Isolation des schémas Postgres » (job `architecture`) rend des **faux positifs**, et il en rendait déjà sur `main` | Son motif `\b[a-z][a-z0-9_]*\.[a-z][a-z0-9_]*` s'applique au fichier ENTIER, commentaires compris. Il accuse donc `verify.sql`, `provision.sql`, `securite.md`, `user.registered` — de la prose — d'être des schémas étrangers, et jusqu'aux séquences `\n` `\r` des chaînes SQL. Vérifié en rejouant le garde sur le contenu de `main` : **9 faux positifs**. Le job `architecture` ne peut donc pas être vert, et ce garde ne garde rien | **Ouvert** — issue #40. Piste : ne matcher que le SQL exécutable. À traiter dans sa propre PR : corriger un motif sans pouvoir exécuter la CI mérite son propre diff |

## Ce que F001 a révélé en se refermant

Rien de tout cela n'était visible tant qu'aucune base n'existait. Les quatre défauts se
succédaient, chacun masqué par le précédent — c'est la forme la plus coûteuse : on croit avoir fini
à chaque étape, et le message d'erreur désigne systématiquement autre chose que la cause.

| # | Défaut | Pourquoi personne ne l'avait vu |
|---|---|---|
| 1 | `verify.sql` vivait **dans** `migrations/postgres/` : goose lit tous les `*.sql` du dossier et refuse ce qui n'est pas une migration | `goose … up` échouait **avant** d'appliquer quoi que ce soit. La CI lance la même commande : le job `migrations` ne pouvait pas passer non plus |
| 2 | `hexa_owner` n'a pas `CREATE` sur la base, donc `CREATE SCHEMA` est refusé | Masqué par le défaut 1, qui échouait plus tôt. Message trompeur : `permission denied for database`, sans nommer de schéma |
| 3 | `DB_SUPERUSER_DSN` visait `postgres` alors que le privilège porte sur la base connectée | Le `GRANT` s'appliquait à la mauvaise base, sans erreur |
| 4 | La migration prend `SET LOCAL ROLE hexa_owner` et ne le relâche pas ; goose écrit sa table de versions **dans la même transaction** | Masqué par le défaut 2. Message encore plus trompeur : `permission denied for table goose_db_version`, une table sans rapport avec le schéma migré — et l'échec survient **après** que tout a été créé, puis tout est annulé |

La leçon n'est pas « il y avait des bugs » : c'est que **du SQL versionné que rien n'exécute est du
SQL qui ne marche pas**, et que la documentation le présentait comme appliqué par la CI. C'est
exactement ce que `rules/README.md` § règle d'or 2 interdit d'écrire.

## Quand écrire une entrée

- On a contourné une règle, même proprement, parce que l'outillage ne permettait pas de la suivre.
- On a passé plus de trente minutes sur un problème qui n'a rien à voir avec le métier.
- On s'est dit « il faudra qu'on regarde ça un jour ».

Ce journal n'est pas une liste de tâches : c'est le relevé des endroits où le cadre coûte plus
qu'il ne rapporte. C'est ce qui permet de le corriger plutôt que de le subir.
