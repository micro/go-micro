//go:generate gorunpkg github.com/99designs/gqlgen

package graphql

import (
	context "context"

	proto "github.com/micro/examples/greeter/srv/proto/hello"
)

type Resolver struct {
	Client proto.SayService
}

func (r *Resolver) Query() QueryResolver {
	return &queryResolver{r}
}

type queryResolver struct{ *Resolver }

func (r *queryResolver) Hello(ctx context.Context, name string) (*proto.Response, error) {
	res, err := r.Client.Hello(ctx, &proto.Request{Name: name})
	if err != nil {
		return &proto.Response{}, err
	}
	return res, nil
}
