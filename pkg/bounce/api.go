package bounce

import (
	"context"

	"moul.io/bounce/pkg/bouncepb"
)

func (svc *Service) Ping(context.Context, *bouncepb.Ping_Request) (*bouncepb.Ping_Response, error) {
	return &bouncepb.Ping_Response{}, nil
}
