package tests

import (
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/modulebus"
)

// TestHTTPModeWithoutAnAddressRefusesToResolve: no address, no call.
//
// # The defect this test catches
//
// Without this guard, the missing address produces a relative URL:
// `"" + "/v1/things"`, that is `/v1/things`. `http.NewRequest` refuses a URL
// with no scheme, so the failure does happen — but AT THE FIRST CALL, in
// production, with a library message talking about a malformed URL. Nobody
// works back from there to "the `interop.base_urls.some_module` variable was
// not defined in the UAT environment".
//
// The guard moves the failure from the first call to START-UP, and the message
// names the module. That is the difference between an incident and a log line.
//
// An address that is present but EMPTY is treated as missing: that is exactly
// what an unresolved `${VAR}` reference produces, and it is the most frequent
// case in practice.
func TestHTTPModeWithoutAnAddressRefusesToResolve(t *testing.T) {
	t.Parallel()

	for name, baseURLs := range map[string]map[string]string{
		"missing table":      nil,
		"empty table":        {},
		"empty entry":        {someModule: ""},
		"a DIFFERENT module": {"other_module": "http://other:8080"},
	} {
		var localCalls int

		_, err := modulebus.Resolve(
			modulebus.New(interop("http", baseURLs), noPublisher(t)),
			someModule, route(), someEvent, localCaller(&localCalls))

		if err == nil {
			t.Errorf("%s: resolution accepted with no address — the failure would happen at the first call", name)
			continue
		}
		if !errors.Is(err, modulebus.ErrNoBaseURL) {
			t.Errorf("%s: returned with %v, want ErrNoBaseURL", name, err)
		}
	}
}
