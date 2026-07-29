package tests

import (
	"net/http"
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/middleware"
)

// TestRecoverAnswers500WithoutLeakingTheStack: a panic becomes a 500 that is
// silent about the details.
//
// Two opposing requirements, and both are needed:
//
//   - The process does not die. Without recovery, ONE panicking request takes
//     the whole server down, hence every in-flight request.
//   - The client learns nothing. A returned call stack exposes file paths,
//     package names and internal structure — the exact map someone probing the
//     service is after.
func TestRecoverAnswers500WithoutLeakingTheStack(t *testing.T) {
	t.Parallel()

	const secret = "password_inside_the_panic"
	panicking := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("internal failure with " + secret)
	})

	rec := call(middleware.Recover(discardLogger()), get(t), panicking)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}

	body := rec.Body.String()
	for _, forbidden := range []string{secret, "goroutine", ".go:", "panic"} {
		if strings.Contains(body, forbidden) {
			t.Errorf("the response holds %q — the cause must never leave: %s", forbidden, body)
		}
	}
}
