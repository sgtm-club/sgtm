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

func (svc *Service) UserList(context.Context, *sgtmpb.UserList_Request) (*sgtmpb.UserList_Response, error) {
	ret := &sgtmpb.UserList_Response{}
	return ret, svc.db.Find(&ret.Users).Error
}

func (svc *Service) PostList(context.Context, *sgtmpb.PostList_Request) (*sgtmpb.PostList_Response, error) {
	ret := &sgtmpb.PostList_Response{}
	return ret, svc.db.Find(&ret.Posts).Error
}
