package generator

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// TrackedFiles enumerates what belongs to the starter.
//
// `git ls-files` rather than a disk walk, and that is not a shortcut: it is the
// only definition that does not go stale. It rules out `.git/`, `bin/`, `.env`,
// `coverage.out` and whatever `.gitignore` will name tomorrow — whereas an
// exclusion list written here would have diverged at the first addition.
//
// Assumed corollary: an untracked file is not copied. That is the right
// behaviour — a file the starter does not version is not part of the starter.
func TrackedFiles(ctx context.Context, root string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "ls-files", "-z")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%s is not a git repository — `hexa new` copies TRACKED "+
			"files, which requires a repository: %w", root, err)
	}

	var files []string
	for name := range strings.SplitSeq(string(out), "\x00") {
		if name != "" {
			files = append(files, name)
		}
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no tracked file in %s: there would be nothing to copy", root)
	}
	return files, nil
}

// CopyProject copies every file, rewriting the module path on the way.
//
// The rewrite happens during the copy, never after: a separate pass over the
// tree would miss files added in between, and above all it would leave an
// intermediate state where the project does not build.
func CopyProject(p ProjectPlan, files []string) error {
	for _, relative := range files {
		if err := copyOne(p, relative); err != nil {
			return err
		}
	}
	return nil
}

// copyOne handles one file, PRESERVING its permissions.
//
// The executable bit is not cosmetic here: `.githooks/commit-msg` and the guards
// in `tools/` do not run without it. This repository has already paid for that
// mistake — its two hooks were versioned as 100644, so git ignored them
// everywhere, on every machine, without anything reporting it.
func copyOne(p ProjectPlan, relative string) error {
	source := filepath.Join(p.Source, relative)
	target := filepath.Join(p.Destination, relative)

	// `git ls-files` never returns a path outside the repository, but asserting
	// that is not enough: it is the kind of invariant that holds until the day
	// somebody calls this function differently. The refusal is explicit.
	if !strings.HasPrefix(target, p.Destination+string(filepath.Separator)) {
		return fmt.Errorf("%s escapes the destination: path refused", relative)
	}

	info, err := os.Lstat(source)
	if err != nil {
		return fmt.Errorf("reading %s: %w", relative, err)
	}
	// A symbolic link copied as a file would change the meaning of the tree. The
	// starter holds none; the day it does, that will have to be decided
	// explicitly rather than discovered after the fact.
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s is a symbolic link: unhandled case, to be decided before versioning it", relative)
	}

	// The path comes from `git ls-files` on the supplied root, never from
	// arbitrary input: it is the starter's own file list.
	content, err := os.ReadFile(source) //nolint:gosec // path coming from git ls-files on the starter root
	if err != nil {
		return fmt.Errorf("reading %s: %w", relative, err)
	}
	if !CitesSocleByHistory(relative) {
		content = []byte(strings.ReplaceAll(string(content), p.SocleModule, p.TargetModule))
	}

	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return fmt.Errorf("creating the directory of %s: %w", relative, err)
	}
	//nolint:gosec // target checked above as internal to the destination
	if err := os.WriteFile(target, content, info.Mode().Perm()); err != nil {
		return fmt.Errorf("writing %s: %w", relative, err)
	}
	return nil
}

// CitesSocleByHistory names the files whose occurrences of the module path are
// LINKS to the starter's history, not imports.
//
// Rewriting them would point the starter's history at a repository that never
// carried it: links to pull requests and issues that do not exist.
//
// The list is identical to the one in `rename:verify` in the Taskfile, and for
// the same reason. It is ENUMERATED, never guessed from a pattern: an exception
// that widens on its own ends up covering a real import.
func CitesSocleByHistory(relative string) bool {
	const separator = "/"
	switch {
	case relative == "documentation/AMORCAGE.md":
		return true
	case relative == "documentation/process/REPRISE.md":
		return true
	case strings.HasPrefix(relative, "documentation/adr"+separator):
		// Every ADR cites its originating issue in its header.
		return true
	default:
		return false
	}
}
