package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestAnHTTPCallCarriesThePayloadAndDecodesTheReply: the remote call is complete.
//
// # What this test locks down, and why each point matters
//
//   - The METHOD and the PATH come from the Route. An ignored route would send
//     a GET where the remote module expects a POST: 405, on every capability at
//     once, and only after a separate deployment.
//   - The body is the serialised request. It is the only thing the remote
//     module receives.
//   - `Content-Type: application/json` is set. Without it, a strict server —
//     and huma is one — refuses the body before having read it.
//   - The base address is joined to the path WITHOUT a doubled slash. An
//     address ending with `/` is what any hand-written configuration produces;
//     `//v1/things` is not the same route to a router.
//   - The reply is decoded into the output type. A call that succeeded while
//     returning the zero value would be the worst of both worlds.
func TestAnHTTPCallCarriesThePayloadAndDecodesTheReply(t *testing.T) {
	t.Parallel()

	var (
		gotMethod      string
		gotPath        string
		gotContentType string
		gotBody        request
	)

	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(reply{Accepted: true})
	}))
	defer remote.Close()

	var localCalls int
	// Address ending with a slash, deliberately.
	cfg := interop("http", map[string]string{someModule: remote.URL + "/"})
	call := resolve(t, cfg, noPublisher(t), localCaller(&localCalls))

	got, err := call(context.Background(), request{Ref: "r-42"})
	if err != nil {
		t.Fatalf("remote call failed: %v", err)
	}

	if gotMethod != "POST" {
		t.Errorf("method received = %q, want POST — the Route is not honoured", gotMethod)
	}
	if gotPath != "/v1/things" {
		t.Errorf("path received = %q, want /v1/things — doubled slash?", gotPath)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type received = %q — a strict server would refuse the body", gotContentType)
	}
	if gotBody.Ref != "r-42" {
		t.Errorf("body received = %+v, the request did not cross over", gotBody)
	}
	if !got.Accepted {
		t.Error("the remote reply was not decoded — the zero value was returned")
	}
	if localCalls != 0 {
		t.Errorf("the local implementation was called %d times in http mode", localCalls)
	}
}
