package tests

import "testing"

// TestVerifyRejectsAWrongPassword: an incorrect password is refused WITHOUT an
// error.
//
// The distinction matters. An error signals a data defect — unreadable digest,
// unknown format — and deserves an alert. A refusal is the nominal case of a
// failed login attempt, and thousands of those happen every day. Confusing them
// would fill the alerts with noise, or mask a real incident.
func TestVerifyRejectsAWrongPassword(t *testing.T) {
	t.Parallel()

	hasher := newHasher()
	encoded := hash(t, "correct horse battery staple")

	for _, wrong := range []string{
		"correct horse battery stapl",  // one character less
		"Correct horse battery staple", // different case
		"",
		"something else entirely",
	} {
		ok, err := hasher.Verify(wrong, encoded)
		if err != nil {
			t.Errorf("Verify(%q) returned an error instead of a refusal: %v", wrong, err)
		}
		if ok {
			t.Errorf("password %q must NOT be accepted", wrong)
		}
	}
}
