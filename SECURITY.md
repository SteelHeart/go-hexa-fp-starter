# Sécurité

## Signaler une faille

**Ne pas ouvrir d'issue publique.** Utiliser l'onglet *Security → Report a vulnerability* du dépôt
(GitHub Private Vulnerability Reporting).

Inclure : ce qui est exploitable, comment le reproduire, l'impact constaté. Une hypothèse est utile,
mais elle doit être annoncée comme telle.

Toute faille retenue devient une entrée `S{NNN}` du
[registre de sécurité](documentation/securite/registre-securite.md), avec son issue et son label
`sec:`. **Une entrée ne se ferme qu'avec son test de non-régression.**

## Ce que ce socle fournit

| Protection | État |
|---|---|
| Hachage de mot de passe Argon2id | Implémenté |
| Chiffrement au repos AES-256-GCM | Implémenté |
| Comparaison de secret en temps constant | Implémenté |
| En-têtes de sécurité HTTP | Implémenté, actif par défaut sur toute route |
| CORS en liste blanche explicite | Implémenté |
| Limitation de débit | Implémenté — **en mémoire, par instance** |
| Taille de corps et délais bornés | Implémenté |
| Récupération de panique sans divulgation | Implémenté |
| Détection de secrets dans l'historique | `gitleaks`, bloquant en CI |
| Vulnérabilités de dépendances | `govulncheck` bloquant + CodeQL hebdomadaire |
| Image non-root, distroless, déployée par digest | Implémenté |

## Ce que ce socle **ne** fournit **pas**

Cette liste existe pour qu'aucune de ces absences ne soit prise pour une protection en place.

- **Aucun mécanisme d'authentification ni d'autorisation.** Le socle fournit les primitives
  cryptographiques, pas l'identité. La feature de référence est publique.
- **Limitation de débit non distribuée** : derrière N répliques, la limite effective est multipliée
  par N.
- **Aucune rotation de clé de chiffrement** : `SECURITY_ENCRYPTION_KEY` est unique et non versionnée.
- **Aucun journal d'audit inaltérable.**
- **Séparation des rôles SQL migration / runtime** : écrite dans les règles, non appliquée en
  développement local.

Périmètre complet : [`documentation/securite/registre-securite.md`](documentation/securite/registre-securite.md)
§ *Ce qui n'est pas encore couvert*.

## Règles applicables

[`rules/securite.md`](rules/securite.md) fait foi. Les points non négociables :

- **Deny par défaut.** Toute garde, toute permission, tout repli sur erreur → refus.
- **Le jeton authentifie, il n'autorise pas.** Les droits fins se vérifient côté serveur, à chaque
  requête, sur l'état persisté.
- **Aucun secret dans le dépôt**, y compris dans `.env.example`. Un secret poussé par erreur est
  **roté**, pas seulement retiré du diff.
- **Aucune donnée personnelle en clair dans les logs.**
- **Aucun contournement de garde CI** — pas de `--no-verify`, pas de seuil desserré.
