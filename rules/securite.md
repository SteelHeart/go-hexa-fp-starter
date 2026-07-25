# Sécurité

Aucune exception, jamais « temporairement ». Une dérogation de sécurité qui survit à la semaine où
elle a été introduite devient permanente.

## 1. Deny par défaut

- Toute garde, toute vérification de droit, **tout repli en cas d'erreur** → refus.
- Une garde qui autorise parce qu'il lui manque du contexte est un défaut critique, pas un cas limite.
- La valeur zéro doit être la valeur sûre : `Result` non initialisé = `Err` ; booléen d'autorisation
  non initialisé = `false` ; config manquante = démarrage refusé.

## 2. Authentification et autorisation

- **Le jeton authentifie, il n'autorise pas.** Les droits fins se vérifient côté serveur, à chaque
  requête, sur l'état persisté.
- L'identité entre dans le cœur comme **paramètre de commande**, jamais lue depuis un contexte
  implicite ni depuis un en-tête.
- Aucune décision d'autorisation dans un adaptateur primaire au-delà de « l'appelant est-il
  authentifié ? ». L'autorisation métier appartient au cas d'usage.
- Toute nouvelle route ajoute sa ligne à la **matrice rôle × endpoint**
  (`documentation/securite/matrice-acces.md`) et son test d'accès refusé.

## 3. Secrets

- **Aucun secret dans le dépôt** — clé, mot de passe, jeton, certificat — y compris dans
  `.env.example`, un fichier de *seed*, un commentaire ou un document. Garde : `gitleaks` en CI, sur
  l'historique complet.
- `.env.example` ne contient que des valeurs **manifestement fausses**, et le commentaire de la
  commande qui génère la vraie valeur.
- Seul `.env.example` est versionné. Tout autre `.env*` est ignoré par `.gitignore`.
- Un secret poussé par erreur est **roté**, pas seulement retiré du diff : il est dans l'historique,
  dans les caches de CI et chez qui a cloné.
- En production, les secrets viennent de l'environnement d'exécution, jamais d'un fichier livré
  avec la source.

## 4. Mots de passe et données sensibles

| Besoin | Règle |
|---|---|
| Mot de passe | **Argon2id** uniquement, paramètres dans `infrastructure/security/`. Jamais bcrypt/SHA/MD5 |
| Comparaison de secret | `subtle.ConstantTimeCompare` — jamais `==` |
| Chiffrement au repos | AES-256-GCM, clé de 32 octets depuis l'environnement, *nonce* aléatoire par message |
| Aléa | `crypto/rand` uniquement. `math/rand` est interdit pour tout ce qui touche à la sécurité |
| Mot de passe en mémoire | type dédié `domain.RawPassword` dont le `String()` retourne `[redacted]` |

## 5. Journalisation

**Ne sont jamais journalisés** : mot de passe (même haché), jeton, clé, cookie de session, numéro
de carte, adresse email complète, donnée de santé, contenu de message.

- Un email se journalise sous forme masquée (`a***@example.com`) ou via son identifiant.
- `domain.Error.Message` est destiné à l'utilisateur ; `cause` est destinée aux logs. Ne jamais
  intervertir : une erreur SQL renvoyée à l'appelant est une fuite de structure interne.
- En cas de doute, journaliser l'**identifiant** plutôt que la **valeur**.

## 6. Surface HTTP

Appliqué par le socle (`internal/pkg/middleware/`), donc actif par défaut sur toute route :

- En-têtes : `Content-Security-Policy`, `X-Content-Type-Options: nosniff`, `Referrer-Policy`,
  `Strict-Transport-Security` (hors développement), `X-Frame-Options: DENY`.
- CORS : **liste blanche explicite** d'origines. Jamais `*` dès qu'il y a une identification.
- Limitation de débit par identité ou par IP, plus stricte sur les routes d'authentification.
- Taille de corps bornée, délais de lecture/écriture bornés, `ReadHeaderTimeout` non nul.
- Récupération de panique qui journalise et renvoie `500` sans divulguer la pile.
- Identifiant de corrélation propagé (`X-Request-Id`) et présent dans chaque log.

## 7. Dépendances

- `govulncheck` bloquant en CI, plus une analyse hebdomadaire (`codeql.yml`) qui attrape les
  vulnérabilités publiées après le dernier merge.
- Toute nouvelle dépendance passe par la procédure de [`dependances.md`](dependances.md).
- Les images sont **distroless, non-root, système de fichiers en lecture seule**, et déployées par
  **digest** `sha256:` — jamais par tag mobile.

## 8. Registre de sécurité

Toute faille identifiée devient une entrée `S{NNN}` dans
[`documentation/securite/registre-securite.md`](../documentation/securite/registre-securite.md),
avec son issue et son label `sec:`.

Une entrée ne se ferme **qu'avec son test de non-régression**. Aucune entrée `sec:critique` ne part
en production ouverte.
