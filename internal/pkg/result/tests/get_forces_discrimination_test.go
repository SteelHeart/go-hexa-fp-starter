package tests

import "testing"

// TestGetForcesDiscrimination: `Get` is the only way out of the box, and it
// returns a boolean.
//
// That boolean is not a convenience, it is a constraint: it forces the call site
// to discriminate. Without it, one could read the value of a failed Result and
// get the zero value of T — exactly the defect Result exists to make impossible.
func TestGetForcesDiscrimination(t *testing.T) {
	t.Parallel()

	value, err, ok := okInt(7).Get()
	if !ok {
		t.Fatal("an Ok must return ok=true")
	}
	if value != 7 {
		t.Errorf("value = %d, want 7", value)
	}
	if err != "" {
		t.Errorf("an Ok must carry no error, got %q", err)
	}

	value, err, ok = errInt("refused").Get()
	if ok {
		t.Fatal("an Err must return ok=false")
	}
	if err != "refused" {
		t.Errorf("error = %q, want \"refused\"", err)
	}
	if value != 0 {
		t.Errorf("an Err must carry no value, got %d", value)
	}
}
