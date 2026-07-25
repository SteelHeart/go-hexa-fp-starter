// Package storage fournit le stockage d'objets derrière un type fonction.
//
// L'adaptateur fourni écrit sur disque : c'est suffisant en développement, et
// remplaçable par S3 sans toucher aux appelants — c'est précisément ce que
// garantit le port.
package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Object est un objet à stocker.
type Object struct {
	Name        string
	ContentType string
	Content     io.Reader
}

// Store écrit un objet et retourne son URL de lecture.
type Store = func(ctx context.Context, obj Object) (string, error)

// Fetch lit un objet stocké.
type Fetch = func(ctx context.Context, key string) (io.ReadCloser, error)

// ErrUnsafeName signale un nom d'objet qui tenterait de sortir du répertoire.
var ErrUnsafeName = errors.New("nom d'objet refusé")

// NewDisk construit un stockage sur disque.
//
// Le nom est haché puis réparti sur deux niveaux de répertoires : un répertoire
// plat à cent mille entrées devient impraticable sur la plupart des systèmes de
// fichiers.
func NewDisk(baseDir, baseURL string) (Store, Fetch, error) {
	if err := os.MkdirAll(baseDir, 0o750); err != nil {
		return nil, nil, fmt.Errorf("création du répertoire de stockage: %w", err)
	}

	store := func(_ context.Context, obj Object) (string, error) {
		key, err := diskKey(obj.Name)
		if err != nil {
			return "", err
		}
		full := filepath.Join(baseDir, filepath.FromSlash(key))
		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			return "", fmt.Errorf("création du sous-répertoire: %w", err)
		}
		file, err := os.OpenFile(full, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o640)
		if err != nil {
			return "", fmt.Errorf("ouverture du fichier de stockage: %w", err)
		}
		defer func() { _ = file.Close() }()
		if _, err := io.Copy(file, obj.Content); err != nil {
			return "", fmt.Errorf("écriture de l'objet: %w", err)
		}
		return strings.TrimSuffix(baseURL, "/") + "/" + key, nil
	}

	fetch := func(_ context.Context, key string) (io.ReadCloser, error) {
		// Nettoyage puis vérification : un `..` dans la clé permettrait de lire
		// n'importe quel fichier de l'hôte.
		clean := path.Clean("/" + key)
		full := filepath.Join(baseDir, filepath.FromSlash(clean))
		if !strings.HasPrefix(full, filepath.Clean(baseDir)+string(os.PathSeparator)) {
			return nil, ErrUnsafeName
		}
		file, err := os.Open(full)
		if err != nil {
			return nil, fmt.Errorf("lecture de l'objet: %w", err)
		}
		return file, nil
	}

	return store, fetch, nil
}

// diskKey dérive une clé de stockage sûre du nom demandé.
func diskKey(name string) (string, error) {
	base := filepath.Base(filepath.Clean(name))
	if base == "." || base == ".." || base == string(os.PathSeparator) {
		return "", ErrUnsafeName
	}
	sum := sha256.Sum256([]byte(name))
	digest := hex.EncodeToString(sum[:])
	return fmt.Sprintf("%s/%s/%s-%s", digest[0:2], digest[2:4], digest[4:16], base), nil
}
