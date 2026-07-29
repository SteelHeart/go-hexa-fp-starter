package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/middleware"
)

// Test_MaxBodyBytes_governs_huma_routes proves that the CONFIGURED bound is the
// one that speaks on a business route — #141.
//
// # What was broken, and why no test caught it
//
// The router mounted `middleware.MaxBody(cfg.HTTP.MaxBodyBytes)`, which works
// perfectly. It simply never got the chance on a huma route:
// `huma.DefaultConfig` carries its OWN limit, nothing linked it to the
// configuration, and it fires first. Every business route goes through huma.
//
// The middleware had its own tests, and they passed. They exercised a bare chi
// route — a path that no business handler takes. **The defect lived exactly in
// the gap between two tested things**, which is where it can live undisturbed.
//
// It was found by MEASURING, not by reading: the P2 persona proof set
// `max_body_bytes` to 50 MiB, posted 5 MiB, and got
// `413 … limit=1048576 bytes` — huma's default, a number written nowhere in
// the configuration.
//
// # Why this test raises the bound instead of lowering it
//
// Raising it is the only direction that can FAIL before the fix. Lowering the
// configured bound below huma's default would be rejected by huma anyway, for
// its own reason, and the test would pass on a broken build — the false green
// this repository hunts.
//
// The second case then lowers it, to prove the configured number governs in
// both directions rather than merely being large enough.
func Test_MaxBodyBytes_governs_huma_routes(t *testing.T) {
	t.Parallel()

	const humaDefault = 1 << 20 // 1 MiB — the value that used to answer

	cases := []struct {
		name       string
		configured int64
		bodySize   int
		wantStatus int
	}{
		{
			// Above huma's default, below the configured bound. This is the
			// case that FAILED before the fix, with 413.
			name:       "a body above huma's default passes when the configuration allows it",
			configured: 4 << 20,
			bodySize:   2 << 20,
			wantStatus: http.StatusOK,
		},
		{
			// Below huma's default: proves the configured bound REFUSES too,
			// and is not simply being overridden by a wider one.
			name:       "a body above the configured bound is refused, even under huma's default",
			configured: 4 << 10,
			bodySize:   64 << 10,
			wantStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name:       "a body under the configured bound passes",
			configured: 4 << 10,
			bodySize:   1 << 10,
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			status := postSizedBody(t, tc.configured, tc.bodySize)
			if status != tc.wantStatus {
				t.Errorf("configured=%d body=%d: got status %d, want %d",
					tc.configured, tc.bodySize, status, tc.wantStatus)
			}
			if tc.wantStatus == http.StatusOK && tc.bodySize > humaDefault && status == http.StatusRequestEntityTooLarge {
				t.Errorf("huma's own %d-byte default is still the one answering — "+
					"humaCfg.MaxBodyBytes is not wired to cfg.HTTP.MaxBodyBytes (#141)", humaDefault)
			}
		})
	}
}

// echoBody is the payload of the throwaway route. A single string field is
// enough: what is measured is the SIZE accepted, not the shape decoded.
type echoBody struct {
	Payload string `json:"payload"`
}

type echoInput struct {
	Body echoBody
}

type echoOutput struct {
	Body echoBody
}

// postSizedBody mounts a huma route on a router built with `configured` as its
// body bound, posts `bodySize` bytes to it, and returns the status.
//
// The route is registered here rather than reusing a business one: this test
// must fail on the ROUTER's wiring, not on a module's validation rules. A
// business route would answer 422 for its own reasons and hide the 413.
func postSizedBody(t *testing.T, configured int64, bodySize int) int {
	t.Helper()

	cfg := serverConfig("production")
	cfg.HTTP.MaxBodyBytes = configured

	rt := routerWith(cfg)

	huma.Register(rt.API, huma.Operation{
		OperationID:  "echo-body",
		Method:       http.MethodPost,
		Path:         "/echo",
		Summary:      "Echo a body — test fixture",
		MaxBodyBytes: middleware.NoBodyLimit,
	}, func(_ context.Context, in *echoInput) (*echoOutput, error) {
		return &echoOutput{Body: in.Body}, nil
	})

	// The JSON envelope adds a few bytes around the payload; the sizes chosen
	// in the cases are far enough from the bounds that this never decides the
	// outcome.
	body := `{"payload":"` + strings.Repeat("x", bodySize) + `"}`

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/echo", strings.NewReader(body))
	req.Header.Set("content-type", "application/json")

	recorder := httptest.NewRecorder()
	rt.Mux.ServeHTTP(recorder, req)

	return recorder.Code
}
