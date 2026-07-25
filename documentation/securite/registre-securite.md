# Registre de sécurité

Toute faille identifiée devient une entrée `S{NNN}`, avec son issue et son label `sec:`.

**Une entrée ne se ferme qu'avec son test de non-régression.** Aucune entrée `sec:critique` ne part
en production ouverte.

| Réf | Date | Sévérité | Description | Statut |
|---|---|---|---|---|
| — | — | — | Aucune faille identifiée à ce jour sur ce socle | — |

## Ce qui n'est pas encore couvert

Cette section liste les protections **absentes**, pour qu'elles ne soient pas confondues avec des
protections en place. Elle n'est pas un registre de failles : c'est le périmètre non traité.

| Sujet | État |
|---|---|
| Authentification et autorisation | **Non implémenté.** Le socle fournit le hachage Argon2id et le chiffrement AES-GCM, pas de mécanisme d'identité. La feature de référence est publique |
| Matrice rôle × endpoint | **Vide** — voir [`matrice-acces.md`](matrice-acces.md) |
| Limitation de débit distribuée | En mémoire par instance uniquement. Derrière plusieurs répliques, la limite effective est multipliée par le nombre d'instances |
| Rotation des clés de chiffrement | Non traitée. `SECURITY_ENCRYPTION_KEY` est une clé unique sans versionnement |
| Audit inaltérable | Non implémenté |
| Séparation des rôles SQL migration / runtime | **Écrite dans les règles, non appliquée** en développement local (même DSN) |

## Format d'une entrée

```
| S001 | 2026-07-25 | sec:critique | Le middleware accepte X-Tenant-Id depuis un en-tête
                                     non authentifié | Ouvert — issue #12 |
```

Champs : référence, date de découverte, sévérité ([`LABELS.md`](../process/LABELS.md)), description
factuelle (ce qui est exploitable, pas ce qu'on suppose), statut et issue.

Une faille **corrigée** garde son entrée, marquée `Fermé — <commit>, test <fichier>`. L'historique
des failles est aussi utile que la liste des failles ouvertes.
