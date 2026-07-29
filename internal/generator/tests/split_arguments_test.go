package tests

import (
	"flag"
	"io"
	"slices"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/generator"
)

// valueFlags builds the option set from a REAL flag set.
//
// The test therefore does not restate the list of options: it derives it exactly
// as the code does. A list written here would end up diverging from the
// commands' — which is precisely what the hand-written table used to do.
func valueFlags(t *testing.T, name string, names ...string) map[string]bool {
	t.Helper()
	set := flag.NewFlagSet(name, flag.ContinueOnError)
	set.SetOutput(io.Discard)
	for _, n := range names {
		set.String(n, "", "")
	}
	return generator.FlagsWithValue(set)
}

// TestSplitArguments: the flag package stops at the FIRST non-option argument.
//
// Without this sorting, `hexa new ./project --module x` would silently ignore
// `--module`, and the command would fail blaming the absence of an option that
// was in fact written — the worst possible message, the one that sends you to
// fix what is already correct.
func TestSplitArguments(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		args           []string
		wantOptions    []string
		wantPositional []string
	}{
		"destination before the option": {
			args:           []string{"./project", "--module", "x/y"},
			wantOptions:    []string{"--module", "x/y"},
			wantPositional: []string{"./project"},
		},
		"destination after the option": {
			args:           []string{"--module", "x/y", "./project"},
			wantOptions:    []string{"--module", "x/y"},
			wantPositional: []string{"./project"},
		},
		"glued form": {
			args:           []string{"./project", "--module=x/y"},
			wantOptions:    []string{"--module=x/y"},
			wantPositional: []string{"./project"},
		},
		"single dash": {
			args:           []string{"-module", "x/y", "./project"},
			wantOptions:    []string{"-module", "x/y"},
			wantPositional: []string{"./project"},
		},
		"two options": {
			args:           []string{"./project", "--module", "x/y", "--from", "/socle"},
			wantOptions:    []string{"--module", "x/y", "--from", "/socle"},
			wantPositional: []string{"./project"},
		},
		"unknown option left to flag": {
			args:           []string{"--unknown", "./project"},
			wantOptions:    []string{"--unknown"},
			wantPositional: []string{"./project"},
		},
	}

	withValue := valueFlags(t, "new", "module", "from")

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			options, positional := generator.SplitArguments(tc.args, withValue)
			if !slices.Equal(options, tc.wantOptions) {
				t.Errorf("options = %v, want %v", options, tc.wantOptions)
			}
			if !slices.Equal(positional, tc.wantPositional) {
				t.Errorf("positional = %v, want %v", positional, tc.wantPositional)
			}
		})
	}
}

// TestEveryDeclaredOptionIsRecognised: the sorting knows the options of EVERY
// command, not only those of `new`.
//
// # The defect this test would have caught
//
// SplitArguments carried a hand-written table: `-module`, `--module`, `-depuis`,
// `--from`. When `make:feature` introduced `--into`, the table did not know
// it — its value was therefore classified as positional, and the command refused
// with:
//
//	flag needs an argument: -dans
//
// A message accusing the user of omitting what they had just written.
//
// The fix was not to add `--into` to the table: it was to REMOVE the table.
// FlagsWithValue derives the options from the flag set, so a declared option is
// recognised by construction. This test checks that on an option which did not
// exist when the sorting was written.
func TestEveryDeclaredOptionIsRecognised(t *testing.T) {
	t.Parallel()

	withValue := valueFlags(t, "make:feature", "into")

	options, positional := generator.SplitArguments(
		[]string{"order_tracking", "--into", "/project"}, withValue)

	if !slices.Equal(options, []string{"--into", "/project"}) {
		t.Errorf("options = %v, want the --into/value pair", options)
	}
	if !slices.Equal(positional, []string{"order_tracking"}) {
		t.Errorf("positional = %v, want only the module name", positional)
	}
}
