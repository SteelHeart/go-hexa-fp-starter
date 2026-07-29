package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// TestAnHTTPFailureNeverLeaksTheRemoteBody: a remote failure stays a status.
//
// # The defect this test catches — and it is a data leak
//
// The reflex, when debugging an inter-module call, is to copy the body of the
// remote reply into the error message: "it helps to diagnose". That body is the
// error message of ANOTHER module, about ITS data. It commonly contains the
// email address that was refused, the customer identifier, the business reason
// for the rejection.
//
// This error travels back to the caller, is logged by it, and goes off to the
// observability sink. Personal data from one module then crosses another one's
// boundary and lands in the logs — exactly what rules/securite.md 5 forbids,
// and by the most mundane path there is.
//
// The rule is therefore: the status, the endpoint, nothing else. A calling
// module has no business knowing another one's internal error taxonomy; it
// translates for itself from the status.
func TestAnHTTPFailureNeverLeaksTheRemoteBody(t *testing.T) {
	t.Parallel()

	// A body that looks like what a real module would return on a 422.
	const secretish = `{"detail":"address alice@example.com already registered","customer":"cus_00042"}`

	for _, status := range []int{
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusUnprocessableEntity,
		http.StatusInternalServerError,
		http.StatusServiceUnavailable,
	} {
		remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(secretish))
		}))

		var localCalls int
		cfg := interop("http", map[string]string{someModule: remote.URL})
		call := resolve(t, cfg, noPublisher(t), localCaller(&localCalls))

		_, err := call(context.Background(), request{Ref: "r-1"})
		remote.Close()

		if err == nil {
			t.Errorf("status %d accepted as a success", status)
			continue
		}
		message := err.Error()

		if strings.Contains(message, "alice@example.com") ||
			strings.Contains(message, "cus_00042") ||
			strings.Contains(message, "already registered") {
			t.Errorf("status %d: the remote body leaked into the error: %s", status, message)
		}
		// The status must be there: without it, the caller cannot translate.
		if !strings.Contains(message, strconv.Itoa(status)) {
			t.Errorf("status %d missing from the message %q — the caller cannot translate", status, message)
		}
	}
}

// TestAnUnreadableReplyIsAFailure: an unreadable body is not a success.
//
// A 200 whose body is not the expected JSON — module of the wrong version,
// proxy inserting an error page — must fail. Returning the zero value would
// make the caller read "refused" on a reply it never received.
func TestAnUnreadableReplyIsAFailure(t *testing.T) {
	t.Parallel()

	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<html>502 Bad Gateway</html>"))
	}))
	defer remote.Close()

	var localCalls int
	cfg := interop("http", map[string]string{someModule: remote.URL})
	call := resolve(t, cfg, noPublisher(t), localCaller(&localCalls))

	if _, err := call(context.Background(), request{Ref: "r-1"}); err == nil {
		t.Error("an unreadable reply was accepted — the caller would read the zero value")
	}
}
