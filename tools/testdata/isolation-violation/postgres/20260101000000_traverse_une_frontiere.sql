-- TÉMOIN — ce fichier DOIT faire échouer le garde d'isolation des schémas.
--
-- Ce n'est pas une migration du dépôt : elle ne sera jamais appliquée, et elle
-- vit hors de `migrations/` précisément pour que goose ne la voie jamais.
--
-- Sa raison d'être est l'ADR 013 : un garde est livré avec le cas qui le fait
-- échouer. Sans ce fichier, on ne saurait pas distinguer « aucune violation »
-- de « le garde ne tourne pas » — et c'est exactement la confusion qui a permis
-- à huit gardes de ce dépôt d'être défectueux sans que personne le voie.
--
-- Le garde doit rendre un code NON NUL ici, en nommant `facturation`.
--
-- ⚠️ Ne pas « corriger » ce fichier. Il est faux exprès.

-- +goose Up
-- +goose StatementBegin

-- Le socle n'a le droit de nommer que `platform` et les schémas système.
-- Celle-ci traverse vers le schéma d'un module métier : c'est la violation.
CREATE TABLE platform.recu (
    id           uuid PRIMARY KEY,
    -- La frontière est franchie ICI, et nulle part ailleurs dans le fichier.
    facture_id   uuid NOT NULL REFERENCES facturation.factures (id)
);

-- Ce commentaire nomme provision.sql et user.registered.v1, et cette chaîne
-- aussi : 'voir facturation.factures'. Aucun des deux ne doit être compté —
-- c'était le défaut de la version précédente du garde.
COMMENT ON TABLE platform.recu IS 'lié à facturation.factures, cf. deploy/postgres/verify.sql';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS platform.recu;
-- +goose StatementEnd
