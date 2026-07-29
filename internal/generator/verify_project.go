package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// VerifyNoTrace refuses a project still carrying the starter's path.
//
// # Why this check is INSIDE the command
//
// A partial rewrite produces a project that builds — Go resolves the import to
// the original starter if it is reachable — but that silently depends on another
// repository. The symptom arrives weeks later, as a package that can no longer
// be found.
//
// This is ADR 013 applied to the generator: it ships with the case that makes it
// fail, and that case is checked on EVERY run rather than once in a test.
//
// Returns the list of offending files rather than logging it: it is the caller —
// the command — that decides how to present it. A library package does not write
// to its caller's error output.
func VerifyNoTrace(p ProjectPlan) ([]string, error) {
	var remaining []string

	err := filepath.WalkDir(p.Destination, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		relative, err := filepath.Rel(p.Destination, path)
		if err != nil {
			return fmt.Errorf("relative path of %s: %w", path, err)
		}
		if CitesSocleByHistory(filepath.ToSlash(relative)) {
			return nil
		}
		content, err := os.ReadFile(path) //nolint:gosec // path coming from the walk of the destination we just wrote
		if err != nil {
			return fmt.Errorf("re-reading %s: %w", relative, err)
		}
		if strings.Contains(string(content), p.SocleModule) {
			remaining = append(remaining, filepath.ToSlash(relative))
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("verifying the generated project: %w", err)
	}
	return remaining, nil
}
