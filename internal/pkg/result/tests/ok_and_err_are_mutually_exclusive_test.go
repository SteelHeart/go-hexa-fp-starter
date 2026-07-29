package tests

import "testing"

// TestOkAndErrAreMutuallyExclusive: a Result carries a success OR an error,
// never both, never neither.
//
// That is what sets it apart from Go's `(T, error)` pair, where nothing prevents
// returning a value AND an error — and where a good share of defects comes from
// the caller reading the value without looking at the error.
func TestOkAndErrAreMutuallyExclusive(t *testing.T) {
	t.Parallel()

	succeeded := okInt(7)
	if !succeeded.IsOk() || succeeded.IsErr() {
		t.Error("an Ok must be IsOk and not IsErr")
	}

	failed := errInt("refused")
	if failed.IsOk() || !failed.IsErr() {
		t.Error("an Err must be IsErr and not IsOk")
	}
}
