package version

import (
	"context"
	"sort"

	"github.com/asim/go-micro/v3/client"
	"github.com/asim/go-micro/v3/selector"
	"github.com/asim/go-micro/v3/registry"
)

// NewClientWrapper is a wrapper which selects only latest versions of services
func NewClientWrapper() client.Wrapper {
	return func(c client.Client) client.Client {
		return &latestVersionWrapper{
			Client: c,
		}
	}
}

type latestVersionWrapper struct {
	client.Client
}

func (w *latestVersionWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	nOpts := append(opts, client.WithSelectOption(selector.WithFilter(filterLatestVersion())))
	return w.Client.Call(ctx, req, rsp, nOpts...)
}

func filterLatestVersion() selector.Filter {
	return func(svcsOld []*registry.Service) []*registry.Service {

		if len(svcsOld) <= 1 {
			return svcsOld
		}

		var svcsNew []*registry.Service
		versions := make([]string, len(svcsOld))

		for i, svc := range svcsOld {
			versions[i] = svc.Version
		}

		sort.Strings(versions)

		gtVersion := versions[len(versions)-1]

		for _, svc := range svcsOld {
			if svc.Version == gtVersion {
				svcsNew = append(svcsNew, svc)
			}
		}

		return svcsNew
	}
}
