// Package disk implements object storage on the local file system.
//
// # NON-GUARANTEES — to be read before using it
//
//   - **No sharing between replicas.** An object written by one instance is
//     nowhere to be found from the others. Behind a load balancer, one upload
//     in two fails on read — this is the non-guarantee that surprises people
//     most.
//   - **No durability beyond the disk.** A container with no mounted volume
//     loses everything on restart.
//   - **The content type is not kept.** That would require a metadata store;
//     the driver has none. A caller that needs it stores it with its own
//     entity.
//   - **No size limit.** It is up to the HTTP surface to set one
//     (`middleware.MaxBody`): the driver writes what it is given.
//   - **No encryption at rest.**
//
// Suitable in development, in test, for a CLI and for any single-instance
// binary with a volume. NOT suitable as soon as there are two replicas.
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

// Permissions of the store.
//
// 0o750 and 0o640: neither read, nor write for "others". An object uploaded by
// a user has no reason to be readable by every account on the machine.
const (
	dirPerm  fs.FileMode = 0o750
	filePerm fs.FileMode = 0o640
)

// Store writes the objects into a directory.
type Store struct {
	baseDir string
	baseURL string
}

// New builds the store and creates its directory.
//
// Fails if the directory cannot be created: discovering an unusable store at
// the first upload by a user would be the worst moment.
func New(baseDir, baseURL string) (*Store, error) {
	if baseDir == "" {
		return nil, errors.New("the disk driver requires a base directory")
	}
	if err := os.MkdirAll(baseDir, dirPerm); err != nil {
		return nil, fmt.Errorf("creating the storage directory %q: %w", baseDir, err)
	}
	return &Store{baseDir: filepath.Clean(baseDir), baseURL: strings.TrimSuffix(baseURL, "/")}, nil
}

// Put implements ports.Put.
func (s *Store) Put(_ context.Context, obj domain.Object) (domain.Located, error) {
	if !obj.IsValid() {
		return domain.Located{}, fmt.Errorf("%w: name=%q", domain.ErrEmptyContent, obj.Name)
	}
	key, err := domain.SafeKey(obj.Name)
	if err != nil {
		return domain.Located{}, fmt.Errorf("deriving the storage key: %w", err)
	}

	full := s.path(key)
	if mkdirErr := os.MkdirAll(filepath.Dir(full), dirPerm); mkdirErr != nil {
		return domain.Located{}, fmt.Errorf("creating the sub-directory: %w", mkdirErr)
	}

	// gosec G304 reports the opening of a variable path, and it is right in
	// general. Here the path does NOT come from `obj.Name`: it is derived by
	// domain.SafeKey, which reduces the name to its last segment and prefixes it
	// with a digest. A traversal (`../../etc/passwd`) does not survive it, and
	// two domain tests lock it down. The validation is upstream, in pure and
	// tested code — not here, in the driver.
	//nolint:gosec // path derived by domain.SafeKey, never supplied by the caller
	file, err := os.OpenFile(full, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, filePerm)
	if err != nil {
		return domain.Located{}, fmt.Errorf("opening the object %s: %w", key, err)
	}
	defer func() { _ = file.Close() }()

	if _, copyErr := io.Copy(file, obj.Content); copyErr != nil {
		return domain.Located{}, fmt.Errorf("writing the object %s: %w", key, copyErr)
	}
	return domain.Located{Key: key, URL: s.baseURL + "/" + key.String()}, nil
}

// Get implements ports.Get.
func (s *Store) Get(_ context.Context, key domain.Key) (io.ReadCloser, error) {
	// The key comes from a URL, hence from a stranger: it is checked BEFORE
	// touching the disk, never after.
	if !domain.IsWithin(key) {
		return nil, fmt.Errorf("%w: key=%q", domain.ErrUnsafeName, key)
	}
	file, err := os.Open(s.path(key))
	if err != nil {
		return nil, fmt.Errorf("reading the object %s: %w", key, err)
	}
	return file, nil
}

// Delete implements ports.Delete.
//
// An object that is already absent is not an error: the caller wants it to be
// gone, and it is gone.
func (s *Store) Delete(_ context.Context, key domain.Key) error {
	if !domain.IsWithin(key) {
		return fmt.Errorf("%w: key=%q", domain.ErrUnsafeName, key)
	}
	if err := os.Remove(s.path(key)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("deleting the object %s: %w", key, err)
	}
	return nil
}

// path resolves the on-disk location of an already validated key.
func (s *Store) path(key domain.Key) string {
	return filepath.Join(s.baseDir, filepath.FromSlash(key.String()))
}
