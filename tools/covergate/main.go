// Command covergate applies the coverage ratchets, identically in local
// development and in CI.
//
// # Why a Go program and not three lines of shell
//
// The thresholds used to live in THREE places, with THREE values:
// `COVER_THRESHOLD: 80` in the Taskfile (never read by any task), `>= 70` in
// CI, `>= 90` for the core. Three sources of truth that contradict one another
// are worth less than a single one that refuses.
//
// A Go program, called by CI AND by `task`, cannot drift: there is only one
// place left where the threshold is written, and it is compiled.
//
// # The defect it fixes
//
// The global ratchet demanded 70 % of a profile produced by `go test ./...`
// WITHOUT A TAG. Yet that batch cannot, by construction, execute a single line
// of a Postgres or Redis driver: `toolchain.md` guarantees that a test without
// a tag requires no service. The threshold was therefore unreachable — not for
// want of rigour, but because it measured code that this batch cannot reach.
//
// An unreachable threshold protects nothing: it turns red permanently, it gets
// ignored, and then it is lowered "to unblock CI". That is exactly the scenario
// rules/tests.md §5 explicitly forbids.
//
// The answer is NOT to lower the bar. It is to measure it where it applies, and
// to state what falls outside. Three ratchets:
//
//  1. UNIT scope      ≥ 70 %   — the code this batch can reach
//  2. business core   ≥ 90 %   — domain/ and application/, weighted
//  3. PRODUCT code    ratchet  — never goes back down
//
// The third one is what stops the exclusion from being a concealment: the
// figure including EVERYTHING is still guarded, and it can only go up. Adding
// uncovered code stays blocked.
package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// The three thresholds. ONE single place in the whole repository.
const (
	// unitThreshold applies to the code that `go test ./...` without a tag can
	// reach.
	unitThreshold = 70.0

	// coreThreshold applies to domain/ and application/ — the business rules.
	// A floor, not a target: coverage measures what is EXECUTED, not what is
	// VERIFIED.
	coreThreshold = 90.0

	// productFloor is a RATCHET, not a target. It bears on ALL the product code
	// — Postgres and Redis drivers included, `cmd/` included — and it never
	// goes down. Raising it is part of every PR that covers code left uncovered
	// until then.
	//
	// This ratchet is what stops the reduced unit scope from being a
	// concealment: the figure including everything stays guarded. Adding
	// uncovered code brings it down, hence fails.
	//
	// History — the ratchet never goes back down:
	//
	//	2026-07-26  measured 56.9 %  →  floor 56
	//	2026-07-27  measured 60.2 %  →  floor 59   (httpserver + telemetry tests)
	//
	// The floor sits about one point below the measurement, to absorb the gap
	// between `-covermode=count` (local) and `atomic` (CI, with -race).
	// Tightening it further is a separate decision: a floor glued to the
	// measurement fails CI over a legitimately removed test, and that is how one
	// ends up disabling the guard rather than honouring it.
	productFloor = 59.0
)

// buildTooling is the ONLY path outside the product-code ratchet.
//
// It is not there because it would matter less, but because a ratchet that
// included the build tooling would punish the act of writing any: adding this
// very tool brought the overall total down from 56.9 % to 54.0 % without a
// single line of product code changing coverage.
const buildTooling = "tools/"

// exclusion names a path outside the UNIT scope, and WHY.
//
// No entry without a written reason: that is what distinguishes an owned scope
// from a circumvented threshold. Every reason must say through which OTHER test
// level the code is supposed to be covered — or admit that none covers it.
type exclusion struct {
	prefix string
	reason string
}

// Shared reasons. Named rather than repeated, so that one copy does not survive
// the others being forgotten.
//
// ⚠️ These reasons used to claim "NO test to this day, at any level (see #37)".
// That became FALSE the day the `integration` level arrived, and a false
// exclusion reason is worse than an exclusion: it justifies a hole that no
// longer exists, and it discourages going to look. Every line below now says
// exactly what covers the package — or what does not cover it.
const (
	reasonNeedsPostgres = "requires Postgres: outside the unit scope, covered by tests/integration (integration tag)"
	reasonNeedsRedis    = "requires Redis: outside the unit scope, covered by tests/integration (integration tag)"
)

//nolint:gochecknoglobals // configuration table of the program, read once
var exclusions = []exclusion{
	{
		prefix: "internal/core/audit/drivers/postgres/",
		reason: reasonNeedsPostgres,
	},
	{
		prefix: "internal/core/dynconf/drivers/postgres/",
		reason: reasonNeedsPostgres,
	},
	{
		prefix: "internal/core/idempotency/drivers/postgres/",
		reason: reasonNeedsPostgres,
	},
	{
		prefix: "internal/core/idempotency/drivers/redis/",
		reason: reasonNeedsRedis,
	},
	{
		prefix: "internal/core/outbox/drivers/postgres/",
		reason: reasonNeedsPostgres,
	},
	{
		prefix: "internal/core/scheduler/drivers/postgres/",
		reason: reasonNeedsPostgres,
	},
	{
		prefix: "internal/infrastructure/database/",
		// The ONLY package in this list that still has NO direct test at all. It
		// is traversed by tests/integration — RunInTx and QuerierFrom are
		// exercised there — but its pool handling, its timeouts and its shutdown
		// are not.
		reason: "opens a pgx pool on construction — traversed by tests/integration, never tested DIRECTLY (#37)",
	},
	{
		prefix: "internal/infrastructure/cache/",
		reason: reasonNeedsRedis,
	},
	{
		prefix: "cmd/",
		reason: "composition root: exercised by tests/e2e (e2e tag), CI e2e job",
	},
	{
		prefix: "tools/",
		reason: "build tooling, not product code — including itself would skew the measurement",
	},
}

// block is a coverage range read from the profile.
type block struct {
	path       string
	statements int
	covered    bool
}

func main() {
	if len(os.Args) < 2 {
		fail("usage: covergate <coverage.out>")
	}
	prefix, err := readModulePrefix("go.mod")
	if err != nil {
		fail(err.Error())
	}
	blocks, err := parseProfile(os.Args[1], prefix)
	if err != nil {
		fail(err.Error())
	}
	if len(blocks) == 0 {
		// False-green trap: an empty profile is not "everything covered".
		fail("EMPTY coverage profile — the test command measured nothing")
	}
	os.Exit(report(blocks))
}

// report applies the three ratchets and returns the exit code.
func report(blocks []block) int {
	failed := false

	if err := checkExclusionsAreAlive(blocks); err != nil {
		fmt.Fprintf(os.Stderr, "\n  FAIL  %s\n", err)
		failed = true
	}

	product := rate(blocks, isProduct)
	unit := rate(blocks, func(b block) bool { return !isExcluded(b.path) })
	core := rate(blocks, isCore)

	fmt.Println("┌─ Coverage ─────────────────────────────────────────────────────")
	printGate("unit scope", unit, unitThreshold, &failed)
	printGate("core (domain + application)", core, coreThreshold, &failed)
	printGate("product code (ratchet)", product, productFloor, &failed)
	fmt.Println("└────────────────────────────────────────────────────────────────")

	printExcluded(blocks)

	if failed {
		return 1
	}
	return 0
}

// printGate prints one ratchet and sets `failed` to true when it does not pass.
//
// ⚠️ The three labels passed in by report are DELIBERATELY left in French: the
// `ci:generateur` task greps them out of the `task check` log of a generated
// project, so that the ratchets of that project show up in the CI output.
// Translating them would silently empty that display, and nothing would say so.
func printGate(name string, got, threshold float64, failed *bool) {
	verdict := "PASS"
	if got < threshold {
		verdict = "FAIL"
		*failed = true
	}
	fmt.Printf("│ %-28s %5.1f %%   threshold %4.1f %%   %s\n", name, got, threshold, verdict)
}

// printExcluded makes visible what has been taken OUT of the unit scope, with
// its real coverage and its reason.
//
// This is the part that stops the exclusion from being a concealment: it prints
// on EVERY run, locally as well as in CI. A reduced scope that is not displayed
// reads as "we cover everything".
func printExcluded(blocks []block) {
	fmt.Println("\nOutside the unit scope — these lines are NOT covered by `go test ./...`:")

	for _, exc := range exclusions {
		matched := filter(blocks, func(b block) bool { return strings.HasPrefix(b.path, exc.prefix) })
		stmts := 0
		for _, b := range matched {
			stmts += b.statements
		}
		fmt.Printf("  %-46s %5.1f %%  %4d instr.  %s\n",
			exc.prefix, rate(matched, func(block) bool { return true }), stmts, exc.reason)
	}
}

// checkExclusionsAreAlive refuses an entry that no longer matches anything.
//
// Without this guard the list ROTS: a renamed package leaves its old entry
// behind, and the scope widens or narrows without anyone having decided it. A
// dead exclusion is an exclusion nobody re-reads any more.
func checkExclusionsAreAlive(blocks []block) error {
	var dead []string
	for _, exc := range exclusions {
		if !contains(blocks, func(b block) bool { return strings.HasPrefix(b.path, exc.prefix) }) {
			dead = append(dead, exc.prefix)
		}
	}
	if len(dead) == 0 {
		return nil
	}
	sort.Strings(dead)
	return fmt.Errorf(
		"exclusion(s) matching no measured code: %s — "+
			"path renamed or removed? drop the entry from tools/covergate",
		strings.Join(dead, ", "))
}

// isCore recognises the business core: the rules, the ones that must hold at
// 90 %.
func isCore(b block) bool {
	return strings.Contains(b.path, "/domain/") || strings.Contains(b.path, "/application/")
}

// isProduct recognises the shipped code, as opposed to the build tooling.
func isProduct(b block) bool {
	return !strings.HasPrefix(b.path, buildTooling)
}

func isExcluded(path string) bool {
	for _, exc := range exclusions {
		if strings.HasPrefix(path, exc.prefix) {
			return true
		}
	}
	return false
}

// rate computes the coverage WEIGHTED BY INSTRUCTION over the kept blocks.
//
// Weighted, and that is a fix: the core ratchet used to average the percentages
// PER FUNCTION. A one-instruction function then weighed as much as a fifty-
// instruction one, so that a handful of covered trivial constructors could mask
// a whole untested business rule.
func rate(blocks []block, keep func(block) bool) float64 {
	total, covered := 0, 0
	for _, b := range blocks {
		if !keep(b) {
			continue
		}
		total += b.statements
		if b.covered {
			covered += b.statements
		}
	}
	if total == 0 {
		return 0
	}
	return 100 * float64(covered) / float64(total)
}

func filter(blocks []block, keep func(block) bool) []block {
	out := make([]block, 0, len(blocks))
	for _, b := range blocks {
		if keep(b) {
			out = append(out, b)
		}
	}
	return out
}

func contains(blocks []block, match func(block) bool) bool {
	for _, b := range blocks {
		if match(b) {
			return true
		}
	}
	return false
}

// parseProfile reads a profile produced by `go test -coverprofile`.
//
// Format of a line, after the `mode:` header:
//
//	<path>:<line>.<col>,<line>.<col> <statement count> <execution count>
func parseProfile(path, modulePrefix string) ([]block, error) {
	file, err := os.Open(path) //nolint:gosec // path supplied by CI or the Taskfile, not by a third party
	if err != nil {
		return nil, fmt.Errorf("reading the profile: %w", err)
	}
	defer func() { _ = file.Close() }()

	// MERGING IS MANDATORY, and this is the central trap of this format.
	//
	// With `-coverpkg=./...`, EVERY test binary emits a profile for ALL the
	// packages of the repository. One and the same range therefore appears as
	// many times as there are test packages — about twenty here — and nearly
	// every one of those occurrences carries a zero counter, since a given
	// binary exercises only a small part of the repository.
	//
	// Summing them naively crushes the measurement: 3.4 % instead of 56.9 %,
	// and absurd statement totals. It is the SAME mistake, in the SAME
	// direction, as the forgotten `-coverpkg` fixed just before: a falsely low
	// figure fails the ratchet for a reason that does not exist, and somebody
	// ends up lowering the threshold.
	//
	// The range is therefore the key, and "covered" is an OR over all of its
	// occurrences: it is enough that ONE binary executed it.
	merged := make(map[string]block)
	scanner := bufio.NewScanner(file)
	for line := 1; scanner.Scan(); line++ {
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" || strings.HasPrefix(raw, "mode:") {
			continue
		}
		key, parsed, err := parseLine(raw, modulePrefix)
		if err != nil {
			return nil, fmt.Errorf("profile line %d: %w", line, err)
		}
		if seen, found := merged[key]; found {
			parsed.covered = parsed.covered || seen.covered
		}
		merged[key] = parsed
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("walking the profile: %w", err)
	}

	blocks := make([]block, 0, len(merged))
	for _, b := range merged {
		blocks = append(blocks, b)
	}
	return blocks, nil
}

// parseLine returns the MERGE KEY (the file and the range, verbatim) and the
// range that was read.
func parseLine(raw, modulePrefix string) (string, block, error) {
	fields := strings.Fields(raw)
	const expectedFields = 3
	if len(fields) != expectedFields {
		return "", block{}, fmt.Errorf("%d fields, %d expected: %q", len(fields), expectedFields, raw)
	}
	statements, err := strconv.Atoi(fields[1])
	if err != nil {
		return "", block{}, fmt.Errorf("unreadable statement count: %q", fields[1])
	}
	count, err := strconv.Atoi(fields[2])
	if err != nil {
		return "", block{}, fmt.Errorf("unreadable execution count: %q", fields[2])
	}
	return fields[0], block{
		path:       modulePath(fields[0], modulePrefix),
		statements: statements,
		covered:    count > 0,
	}, nil
}

// modulePath reduces the profile path to a path relative to the repository.
//
// The profile names files by their full IMPORT path
// (`github.com/…/internal/config/config.go:12.5,14.3`). The exclusions, on the
// other hand, are written as repository paths: that is what a reviewer
// recognises.
//
// # The prefix is READ from go.mod, never written here
//
// It used to be written here: `const marker = "go-hexa-fp-starter/"`. The
// defect only surfaced at the first use of `hexa new` — in a generated project
// that marker matches nothing, NO exclusion applies, and the three ratchets
// measure a scope that is not theirs. The first two dropped to 56.7 % against
// 74.8 % here.
//
// Two guards existed and could not see it: `task rename` and its
// `rename:verify` look for the COMPLETE module path, whereas this constant
// carried only its last segment. A binding by substring escapes any
// search-and-replace — which is what makes it durable.
func modulePath(raw, modulePrefix string) string {
	path := raw
	if colon := strings.LastIndex(path, ":"); colon >= 0 {
		path = path[:colon]
	}
	return strings.TrimPrefix(path, modulePrefix)
}

// readModulePrefix reads the module path of the current repository, terminated
// by `/`.
//
// Returning an error rather than a default: without this prefix the exclusions
// match nothing and the ratchets measure wrong. A tool that guessed here would
// fail silently, in the "too low" direction — the one that lowers a threshold
// for a reason that does not exist.
func readModulePrefix(path string) (string, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // fixed name, resolved from the current directory
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if rest, found := strings.CutPrefix(strings.TrimSpace(line), "module "); found {
			return strings.TrimSpace(rest) + "/", nil
		}
	}
	return "", fmt.Errorf("no `module` directive in %s", path)
}

func fail(message string) {
	fmt.Fprintf(os.Stderr, "covergate: %s\n", message)
	os.Exit(1)
}
