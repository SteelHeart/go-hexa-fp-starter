package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────────
// The hardening nobody would have noticed missing
// ─────────────────────────────────────────────────────────────────────────────
//
// `arch-go.yml` forbids a business module from importing another one — sealing
// between contexts (ADR 011). But the rule NAMES the module explicitly:
//
//	- package: "**.internal.modules.user_registration.**"
//	  shouldNotDependsOn:
//	    internal:
//	      - "**.internal.modules.!(user_registration).**"
//
// A module created without touching it is covered by NO sealing rule at all. It
// could import any other module, and `arch-go` would report "100% compliance" —
// because it has nothing to say about a module nobody told it about.
//
// This is precisely the shape of defect this repository has paid for eleven
// times: a mute guard is indistinguishable from a satisfied one.
// `hexa make:feature` therefore writes the rule at the same time as the module,
// and REFUSES if it cannot write it — rather than silently creating an unguarded
// module.

// IsolationAnchor locates where to insert the sealing rule.
//
// A type rather than three returns: the two-returns rule holds here too.
type IsolationAnchor struct {
	path    string
	content string
	// after is the END index of the last existing sealing rule.
	after int
}

// isolationPattern recognises the last line of a sealing rule.
//
// The anchoring rests on the SHAPE of the rule, not on a comment nor on a
// section number: those move at the first reshuffle of the file, the shape does
// not. Should it move anyway, the command refuses — it does not guess.
var isolationPattern = regexp.MustCompile(`(?m)^\s+- "\*\*\.internal\.modules\.!\([a-z0-9_]+\)\.\*\*"$`)

// FindIsolationAnchor reads `arch-go.yml` and locates the insertion point.
func FindIsolationAnchor(root, dir string) (IsolationAnchor, error) {
	path := filepath.Join(root, "arch-go.yml")
	// Root named by the caller, fixed file name.
	//nolint:gosec // root supplied by the user, constant file name
	raw, err := os.ReadFile(path)
	if err != nil {
		return IsolationAnchor{}, fmt.Errorf(
			"%s is unreadable — is this really the root of a project made from the starter? %w", path, err)
	}
	content := string(raw)

	if strings.Contains(content, "internal.modules."+dir+".**") {
		return IsolationAnchor{}, fmt.Errorf(
			"arch-go.yml already declares a rule for %q — does the module exist elsewhere?", dir)
	}

	positions := isolationPattern.FindAllStringIndex(content, -1)
	if len(positions) == 0 {
		return IsolationAnchor{}, fmt.Errorf(
			"no sealing rule in %s: cannot add the one for %q.\n"+
				"       Deliberate refusal — creating a module NO rule guards would be worse\n"+
				"       than not creating it. Add it by hand, under `dependenciesRules:`:\n\n%s",
			path, dir, IsolationRule(dir))
	}

	last := positions[len(positions)-1]
	return IsolationAnchor{path: path, content: content, after: last[1]}, nil
}

// IsolationRule returns the YAML sealing block of a module.
func IsolationRule(dir string) string {
	return fmt.Sprintf(`  - package: "**.internal.modules.%s.**"
    shouldNotDependsOn:
      internal:
        - "**.internal.modules.!(%s).**"
`, dir, dir)
}

// DeclareIsolation inserts the rule into `arch-go.yml`.
//
// The write preserves the rest of the file byte for byte: `arch-go.yml` is
// almost entirely made of comments explaining WHY each rule exists, and a YAML
// round trip would wipe them all.
func DeclareIsolation(a IsolationAnchor, dir string) error {
	merged := a.content[:a.after] + "\n\n" + strings.TrimRight(IsolationRule(dir), "\n") + a.content[a.after:]

	if err := os.WriteFile(a.path, []byte(merged), 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", a.path, err)
	}
	return nil
}
