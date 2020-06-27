package sgtm

import (
	"context"
	"fmt"
	"os"
	"time"

	"moul.io/godev"
	"moul.io/sgtm/pkg/sgtmpb"
)

func (svc *Service) Register(ctx context.Context, input *sgtmpb.Register_Request) (*sgtmpb.Register_Response, error) {
	fmt.Println(godev.PrettyJSON(input))
	if input == nil || input.Email == "" || input.Username == "" {
		return nil, fmt.Errorf("missing required fields")
	}

	// FIXME: generate username if empty
	// FIXME: captcha
	// FIXME: validate valid email
	// FIXME: activity -> register (in a transaction)
	user := sgtmpb.User{
		Username:  input.Username,
		Email:     input.Email,
		Firstname: input.Firstname,
		Lastname:  input.Lastname,
	}
	err := svc.db.Create(&user).Error
	if err != nil {
		return nil, fmt.Errorf("db.Create: %w", err)
	}

	ret := sgtmpb.Register_Response{
		User: &user,
	}
	// FIXME: send an email
	return &ret, nil
}

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
