# Matrice rôle × endpoint

Toute route ajoute sa ligne ici, **et** son test d'accès refusé
([`rules/securite.md`](../../rules/securite.md) §2).

Une case vide est interdite : si le droit n'est pas tranché, l'issue porte `needs-decision` et la
route n'est pas livrée.

## Légende

| Symbole | Sens |
|---|---|
| ✅ | Autorisé |
| ❌ | Refusé — **et un test le prouve** |
| 🔓 | Public, sans authentification — décision explicite, jamais un défaut |

## HTTP

| Méthode | Route | Anonyme | Utilisateur | Administrateur | Test de refus |
|---|---|---|---|---|---|
| `POST` | `/v1/users` | 🔓 | 🔓 | 🔓 | — (route d'inscription, publique par conception) |
| `GET` | `/v1/users/email-availability` | 🔓 | 🔓 | 🔓 | — (route de vérification, publique par conception) |
| `GET` | `/healthz` | 🔓 | 🔓 | 🔓 | — |
| `GET` | `/readyz` | 🔓 | 🔓 | 🔓 | — |
| `GET` | `/metrics` | ❌ | ❌ | ❌ | port séparé, non exposé publiquement |

> Les deux routes publiques ci-dessus sont des **surfaces d'énumération** : elles permettent de
> tester si un email est enregistré. C'est un compromis assumé pour une inscription utilisable, et
> il impose une **limitation de débit stricte** sur ces routes — c'est la seule mitigation en
> place. À réévaluer dès qu'un mécanisme d'identité existe.

## CLI

| Commande | Contexte d'exécution | Autorisation |
|---|---|---|
| `cli register` | Opérateur sur la machine | L'accès au binaire et à la base **est** l'autorisation. Aucune vérification supplémentaire |

> Conséquence à ne pas perdre de vue : quiconque peut exécuter le binaire peut créer un compte. La
> CLI est un outil d'administration, elle n'est pas destinée à un utilisateur final.

## Événements

| Type d'événement | Producteur autorisé | Consommateur |
|---|---|---|
| `user.registered.v1` | `user_registration` uniquement | worker → envoi du courriel de bienvenue |

L'outbox est interne : aucun producteur externe n'y écrit.
