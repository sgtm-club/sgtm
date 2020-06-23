package sgtm

import (
	"context"
	"os"
	"time"

	"moul.io/sgtm/pkg/sgtmpb"
)

func (svc *Service) Ping(context.Context, *sgtmpb.Ping_Request) (*sgtmpb.Ping_Response, error) {
	return &sgtmpb.Ping_Response{}, nil
}

func (svc *Service) Status(context.Context, *sgtmpb.Status_Request) (*sgtmpb.Status_Response, error) {
	hostname, _ := os.Hostname()
	return &sgtmpb.Status_Response{
		Uptime:   int32(time.Since(svc.startedAt).Seconds()),
		Hostname: hostname,
	}, nil
}
