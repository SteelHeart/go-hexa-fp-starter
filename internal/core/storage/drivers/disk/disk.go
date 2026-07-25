// Package disk implémente le stockage d'objets sur le système de fichiers local.
//
// # NON-GARANTIES — à lire avant de l'utiliser
//
//   - **Aucun partage entre répliques.** Un objet écrit par une instance est
//     introuvable depuis les autres. Derrière un répartiteur, un téléversement sur
//     deux échoue à la lecture — c'est la non-garantie qui surprend le plus.
//   - **Aucune durabilité au-delà du disque.** Un conteneur sans volume monté
//     perd tout au redémarrage.
//   - **Le type de contenu n'est pas conservé.** Il faudrait un magasin de
//     métadonnées ; le pilote n'en a pas. Un appelant qui en a besoin le stocke
//     avec sa propre entité.
//   - **Aucune limite de taille.** C'est à la surface HTTP de la poser
//     (`middleware.MaxBody`) : le pilote écrit ce qu'on lui donne.
//   - **Aucun chiffrement au repos.**
//
// Convient en développement, en test, pour une CLI et pour tout binaire
// mono-instance avec un volume. Ne convient PAS dès qu'il y a deux répliques.
package disk

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/storage/domain"
)

// Permissions du magasin.
//
// 0o750 et 0o640 : ni lecture, ni écriture pour « les autres ». Un objet
// téléversé par un utilisateur n'a aucune raison d'être lisible par tout compte
// de la machine.
const (
	dirPerm  fs.FileMode = 0o750
	filePerm fs.FileMode = 0o640
)

// Store écrit les objets dans un répertoire.
type Store struct {
	baseDir string
	baseURL string
}

// New construit le magasin et crée son répertoire.
//
// Échoue si le répertoire n'est pas créable : découvrir un magasin inutilisable au
// premier téléversement d'un utilisateur serait le pire moment.
func New(baseDir, baseURL string) (*Store, error) {
	if baseDir == "" {
		return nil, errors.New("le pilote disk exige un répertoire de base")
	}
	if err := os.MkdirAll(baseDir, dirPerm); err != nil {
		return nil, fmt.Errorf("création du répertoire de stockage %q: %w", baseDir, err)
	}
	return &Store{baseDir: filepath.Clean(baseDir), baseURL: strings.TrimSuffix(baseURL, "/")}, nil
}

// Put implémente ports.Put.
func (s *Store) Put(_ context.Context, obj domain.Object) (domain.Located, error) {
	if !obj.IsValid() {
		return domain.Located{}, fmt.Errorf("%w: nom=%q", domain.ErrEmptyContent, obj.Name)
	}
	key, err := domain.SafeKey(obj.Name)
	if err != nil {
		return domain.Located{}, err
	}

	full := s.path(key)
	if err := os.MkdirAll(filepath.Dir(full), dirPerm); err != nil {
		return domain.Located{}, fmt.Errorf("création du sous-répertoire: %w", err)
	}

	file, err := os.OpenFile(full, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, filePerm)
	if err != nil {
		return domain.Located{}, fmt.Errorf("ouverture de l'objet %s: %w", key, err)
	}
	defer func() { _ = file.Close() }()

	if _, err := io.Copy(file, obj.Content); err != nil {
		return domain.Located{}, fmt.Errorf("écriture de l'objet %s: %w", key, err)
	}
	return domain.Located{Key: key, URL: s.baseURL + "/" + key.String()}, nil
}

// Get implémente ports.Get.
func (s *Store) Get(_ context.Context, key domain.Key) (io.ReadCloser, error) {
	// La clé vient d'une URL, donc d'un inconnu : elle est vérifiée AVANT de
	// toucher au disque, jamais après.
	if !domain.IsWithin(key) {
		return nil, fmt.Errorf("%w: clé=%q", domain.ErrUnsafeName, key)
	}
	file, err := os.Open(s.path(key))
	if err != nil {
		return nil, fmt.Errorf("lecture de l'objet %s: %w", key, err)
	}
	return file, nil
}

// Delete implémente ports.Delete.
//
// Un objet déjà absent n'est pas une erreur : l'appelant veut qu'il ne soit plus
// là, et il n'y est plus.
func (s *Store) Delete(_ context.Context, key domain.Key) error {
	if !domain.IsWithin(key) {
		return fmt.Errorf("%w: clé=%q", domain.ErrUnsafeName, key)
	}
	if err := os.Remove(s.path(key)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("suppression de l'objet %s: %w", key, err)
	}
	return nil
}

// path résout l'emplacement sur disque d'une clé déjà validée.
func (s *Store) path(key domain.Key) string {
	return filepath.Join(s.baseDir, filepath.FromSlash(key.String()))
}
