package generator

import (
	"flag"
	"strings"
)

// FlagsWithValue returns a set's options, in both spellings.
//
// Derived from the FlagSet rather than written by hand, and that is the point: a
// hand-written table listed `-module` and `-from`. The `--into` option added by
// `make:feature` was not in it, so its value was classified as positional and
// the command refused an option that had actually been written. A second list of
// the same options can only diverge from the first.
func FlagsWithValue(set *flag.FlagSet) map[string]bool {
	withValue := map[string]bool{}
	set.VisitAll(func(f *flag.Flag) {
		withValue["-"+f.Name] = true
		withValue["--"+f.Name] = true
	})
	return withValue
}

// SplitArguments puts options on one side and positionals on the other.
//
// The `flag` package stops at the FIRST non-option argument: without this
// sorting, `hexa new ./project --module x` would silently ignore `--module`, and
// the command would fail blaming the absence of an option that was written.
//
// Forcing the `--module x ./project` order would be gratuitous friction in a
// tool whose whole purpose is to remove it.
func SplitArguments(args []string, withValue map[string]bool) (options, positional []string) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case !strings.HasPrefix(arg, "-"):
			positional = append(positional, arg)
		case withValue[arg] && i+1 < len(args):
			// Form `--module x`: the value follows, it is not positional.
			options = append(options, arg, args[i+1])
			i++
		default:
			// Form `--module=x`, or an unknown option `flag` will refuse itself.
			options = append(options, arg)
		}
	}
	return options, positional
}
