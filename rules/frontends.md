# Frontends — N surfaces sur un seul cœur

> Décision de référence : [ADR 005](../documentation/adr/005-n-frontends-adaptateurs-primaires.md).

## 1. Le principe

**Un frontend est un adaptateur primaire. Rien d'autre.** Web, mobile, CLI, consommateur
d'événements, gRPC, tâche planifiée : tous appellent **la même fonction de cas d'usage**.

```
        ┌──────────── adapters/primary/ ────────────┐
 web ──►│ http/    handler huma                     │─┐
mobile ►│ http/    (même route, autre présentateur)  │ │
 CLI ──►│ cli/     parse des flags                   │ ├──► ports.RegisterUser
events ►│ events/  consomme un message               │ │    (une seule implémentation)
 gRPC ─►│ grpc/    à ajouter le jour où c'est utile  │─┘
        └───────────────────────────────────────────┘
```

Corollaire mesurable : **ajouter un frontend ne modifie aucun fichier de `domain/`, `ports/` ou
`application/`.** Si une PR qui ajoute une surface touche ces dossiers, la conception est fausse.

## 2. Obligations d'un adaptateur primaire

| Il fait | Il ne fait jamais |
|---|---|
| Parser l'entrée du transport (JSON, flags, payload) | Contenir une règle métier |
| Construire la `Command` du domaine | Accéder à la base de données |
| Appeler **un** port primaire | Construire ses dépendances |
| Traduire le `Result` dans les codes du transport | Journaliser une donnée personnelle |
| Fixer les délais et limites propres au transport | Appeler un autre adaptateur primaire |

Il reçoit les ports **par paramètre** — jamais un `*pgxpool.Pool`, jamais une config.

```go
// adapters/primary/http/handler.go
func Mount(api huma.API, register ports.RegisterUser) { … }

// adapters/primary/cli/command.go
func NewRegisterCommand(register ports.RegisterUser) Command { … }
```

## 3. Présentateurs : jamais de DTO partagé entre surfaces

Web, mobile et CLI n'ont pas les mêmes besoins de forme : le mobile veut peu d'octets, le web veut
des libellés, la CLI veut du texte alignable. Un DTO partagé fige les trois ensemble et rend toute
évolution cassante pour tous.

**Chaque adaptateur possède ses propres types de sortie**, dans son propre paquet. La duplication
apparente est voulue ([`solid-et-dry.md`](solid-et-dry.md) § *Ce qui doit rester dupliqué*).

## 4. Un frontend, un binaire ? Non : un binaire, N frontends

`cmd/server` monte **plusieurs** adaptateurs primaires simultanément (HTTP public, HTTP interne,
consommateur d'événements) sur les **mêmes** modules. Le découpage en binaires suit le **cycle de
vie opérationnel**, pas le nombre de frontends :

| Binaire | Rôle | Pourquoi séparé |
|---|---|---|
| `cmd/server` | HTTP (web + mobile) | Se met à l'échelle sur le trafic entrant |
| `cmd/worker` | dépilage outbox, tâches | Se met à l'échelle sur la profondeur de file ; ne doit pas mourir avec le serveur |
| `cmd/cli` | administration, scripts, CI | Doit démarrer en millisecondes, sans écouter de port |

## 5. Authentification par surface

Le cœur ne connaît **aucun** mécanisme d'authentification : il reçoit une identité déjà établie
dans un paramètre de commande.

| Surface | Mécanisme | Établi par |
|---|---|---|
| Web | cookie de session `HttpOnly` + `SameSite` | middleware HTTP |
| Mobile | jeton porteur à durée courte + rafraîchissement | middleware HTTP |
| CLI | clé d'API à portée limitée, ou identité de la machine | adaptateur CLI |
| Événements | identité du producteur, portée par l'enveloppe du message | adaptateur events |

**Le jeton authentifie, il n'autorise pas.** Les droits fins se vérifient côté serveur, à chaque
appel, sur la base de l'état persisté — jamais sur un *claim* de jeton
([`securite.md`](securite.md)).

## 6. Ajouter une surface — la procédure

1. Créer `internal/modules/{module}/adapters/primary/{transport}/`.
2. Écrire le parsing d'entrée, l'appel du port, la traduction du `Result`. **Rien d'autre.**
3. Ajouter son propre présentateur. Ne pas réutiliser celui d'une autre surface.
4. Monter l'adaptateur dans le `cmd/*` concerné (ou en créer un nouveau si le cycle de vie diffère).
5. Traiter **exhaustivement** les `domain.ErrorCode` : le linter `exhaustive` échoue sinon.
6. Test de bout en bout de la nouvelle surface sur au moins un chemin nominal et un chemin d'erreur.

Si l'étape 2 déborde, c'est que la logique appartient au cœur : elle y remonte, elle ne se
duplique pas dans la surface.

## 7. Clients générés

`api/openapi.yaml` est généré depuis le code ; les SDK clients sont générés depuis ce fichier
(`task sdk:ts`, et l'équivalent Dart/Swift à ajouter selon les surfaces réelles).

Un frontend qui écrit ses types d'API à la main est en dette dès la PR suivante. Cette règle
n'est pas outillable depuis ce dépôt (`[humain]`) : elle relève de la revue côté frontend.
