package bounce

import (
	"context"
	"os"
	"time"

	"moul.io/bounce/pkg/bouncepb"
)

func (svc *Service) Ping(context.Context, *bouncepb.Ping_Request) (*bouncepb.Ping_Response, error) {
	return &bouncepb.Ping_Response{}, nil
}

func (svc *Service) Status(context.Context, *bouncepb.Status_Request) (*bouncepb.Status_Response, error) {
	hostname, _ := os.Hostname()
	return &bouncepb.Status_Response{
		Uptime:   int32(time.Since(svc.startedAt).Seconds()),
		Hostname: hostname,
	}, nil
}
