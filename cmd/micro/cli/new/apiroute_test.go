package new

import (
	"flag"
	"fmt"
	"github.com/urfave/cli/v2"
	"net/http"
	"testing"
)

type apiRouteCredentialTransport struct {
	t      *testing.T
	called bool
}

func (r *apiRouteCredentialTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r.called = true
	if got := req.Header.Get("Authorization"); got != "Bearer route-key" {
		r.t.Errorf("authorization = %q", got)
	}
	if req.URL.Host != "global.api-route.com" {
		r.t.Errorf("host = %s", req.URL.Host)
	}
	return nil, fmt.Errorf("stop before network")
}
func TestAPIRoutePromptKey(t *testing.T) {
	t.Setenv("MICRO_AI_API_KEY", "")
	t.Setenv("API_ROUTE_API_KEY", "route-key")
	t.Setenv("ANTHROPIC_API_KEY", "other-provider-key")
	transport := &apiRouteCredentialTransport{t: t}
	old := http.DefaultClient
	http.DefaultClient = &http.Client{Transport: transport}
	t.Cleanup(func() { http.DefaultClient = old })
	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	flags.String("provider", "api-route", "")
	flags.String("api_key", "", "")
	err := runPrompt(cli.NewContext(cli.NewApp(), flags, nil), "create a service")
	if err == nil || !transport.called {
		t.Fatalf("provider not reached: %v", err)
	}
}
