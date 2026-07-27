# Toolbox — l'outillage en conteneur, rien sur le poste

> **Ce que ce dossier résout** : `rules/toolchain.md` §3 suppose Go et Task installés sur la
> machine, et `task tools` ajoute quatre binaires dans le `GOPATH` de l'utilisateur. Sur un poste
> où l'on ne veut **rien** installer, ce n'est pas une gêne mais un blocage total : sans Go, aucune
> des six étapes de `task check` ne démarre.
>
> La toolbox déplace l'outillage dans une image. Le dépôt est monté, les caches Go vivent dans des
> volumes, et le poste ne reçoit qu'un script `sh` de cinquante lignes.

## Les trois commandes

```bash
./deploy/toolbox/tb                # un shell dans la toolbox
./deploy/toolbox/tb task check     # fmt · vet · lint · arch · test · vuln
./deploy/toolbox/tb task ci        # la barrière COMPLÈTE — 10 des 12 jobs de la CI
./deploy/toolbox/tb task up        # démarre la pile — via le moteur de l'HÔTE
```

Rien d'autre à connaître : **toute** commande du dépôt se préfixe par `tb`.

### `task check` ou `task ci` ?

`task check` couvre **4** des 12 jobs de la CI. `task ci` en couvre **10** — il y ajoute les
migrations avec leur `Down` réellement exécuté, le niveau `e2e` avec les tests **comptés**, la
compilation croisée sur les quatre cibles, `gitleaks` sur l'historique complet, et la construction
des deux images livrées. Les deux restants dépendent du contexte d'une PR et se lancent à part :

```bash
./deploy/toolbox/tb task ci:pr-title -- "fix(data): un titre conventionnel"
./deploy/toolbox/tb task ci:inertia  -- origin/main
```

> ⚠️ **`task ci` n'est pas une garde.** Elle s'exécute sur la machine de l'auteur, qui peut ne pas
> la lancer. C'est un **instrument de mesure reproductible**, pas un contrôle — la même distinction
> qu'entre le crochet local et le ruleset serveur (`rules/toolchain.md` §4). Tant que la CI ne
> démarre pas, fusionner reste une **décision documentée**, jamais un gate franchi.

## Ce qui rend `task up` possible depuis l'intérieur

La toolbox ne contient pas de moteur de conteneurs : elle **pilote celui de l'hôte** par sa socket,
montée sur `/var/run/docker.sock`. Podman expose une API compatible Docker et la sous-commande
`compose` est identique chez les deux — `docker compose up -d --wait`, la ligne écrite dans le
`Taskfile`, fonctionne donc telle quelle sans qu'aucune variante Podman ait à y être ajoutée.

Prérequis côté poste, une seule fois :

```bash
systemctl --user enable --now podman.socket
```

Sans elle, tout le reste fonctionne ; seules les tâches de pile locale échouent, en le disant.

## Ce que l'image apporte, et à quelle friction

| Contenu | Ce que ça débloque |
|---|---|
| `go1.25.12` | **Exactement** la version épinglée par `go.mod`. `GOTOOLCHAIN` n'a rien à télécharger — F007 ne peut pas revenir par la bande |
| `gcc` + `musl-dev` | CGO, donc `-race`. **F005** : le détecteur de courses ne tournait qu'en CI |
| `psql` 17 | `task db:provision` et `task db:verify`. **F001** : les invariants de l'ADR 011 n'étaient vérifiables qu'en CI |
| `goose` | `task migrate:*`, donc les migrations exécutées en local pour de vrai |
| `docker-cli` + `compose` | `task up`, `task down`, `task logs`, sans toucher au `Taskfile` |
| `task`, `golangci-lint`, `arch-go`, `govulncheck` | les six étapes de `task check` |

## Réseau, fichiers, caches

- **`--network host`** : les ports publiés par la pile se voient à `localhost`, donc `.env` et
  `config/*.yaml` sont valables **à l'identique** dedans et dehors. Et le serveur lancé par
  `task run` reste joignable depuis le poste — d'où le `curl localhost:8080` habituel.
- **Rootless** : l'UID 0 du conteneur *est* l'utilisateur de l'hôte. Les fichiers produits dans le
  dépôt (`coverage.out`, `bin/`) lui appartiennent normalement — aucun `chown` à faire après coup.
- **Caches** : `hexa-toolbox-gomod` et `hexa-toolbox-gobuild` survivent à la reconstruction de
  l'image. Sans eux, chaque exécution retéléchargerait le graphe de modules.

## Reconstruire

```bash
./deploy/toolbox/tb --rebuild
```

## Ce que la toolbox ne résout PAS

- **F004 — reproductibilité.** Les quatre outils Go sont en `latest`, comme dans `Taskfile.yml`.
  L'image les fige *entre deux postes* qui partagent le même identifiant d'image, mais deux
  constructions à un mois d'écart peuvent différer. Les `ARG` du `Containerfile` existent pour que
  le gel (première release, `rules/dependances.md` §5) soit un changement d'une ligne.
- **Le `Dockerfile` de la racine.** Il produit l'image **livrée** — distroless, non-root, sans
  shell. La toolbox est un environnement de travail et n'est jamais déployée. Ne pas confondre les
  deux, ni les fusionner : leurs surfaces d'attaque n'ont rien à voir.
