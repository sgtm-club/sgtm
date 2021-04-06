package sgtm

import (
	"context"
	"fmt"
	"os"
	"time"

	"google.golang.org/grpc/metadata"
	"moul.io/sgtm/pkg/sgtmpb"
)

func (svc *Service) claimsFromContext(ctx context.Context) (*jwtClaims, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("cannot get metadata from context")
	}

	oauthToken, ok := md["oauth-token"]
	if !ok || len(oauthToken) == 0 {
		return nil, fmt.Errorf("no such oauth-token")
	}

	return svc.parseJWTToken(oauthToken[0])
}

func (svc *Service) Me(ctx context.Context, req *sgtmpb.Me_Request) (*sgtmpb.Me_Response, error) {
	claims, err := svc.claimsFromContext(ctx)
	if err != nil {
		return nil, err
	}

	user, err := svc.storage.GetMe(claims.Session.UserID)
	if err != nil {
		return nil, err
	}
	return &sgtmpb.Me_Response{User: user}, nil

}

func (svc *Service) Ping(context.Context, *sgtmpb.Ping_Request) (*sgtmpb.Ping_Response, error) {
	return &sgtmpb.Ping_Response{}, nil
}

func (svc *Service) Status(context.Context, *sgtmpb.Status_Request) (*sgtmpb.Status_Response, error) {
	hostname, _ := os.Hostname()
	return &sgtmpb.Status_Response{
		Uptime:         int32(time.Since(svc.StartedAt).Seconds()),
		Hostname:       hostname,
		EverythingIsOk: true,
	}, nil
}

func (svc *Service) UserList(context.Context, *sgtmpb.UserList_Request) (*sgtmpb.UserList_Response, error) {
	users, err := svc.storage.GetUsersList()
	if err != nil {
		return nil, err
	}
	return &sgtmpb.UserList_Response{Users: users}, nil
}

func (svc *Service) PostList(context.Context, *sgtmpb.PostList_Request) (*sgtmpb.PostList_Response, error) {
	posts, err := svc.storage.GetPostList()
	if err != nil {
		return nil, err
	}

	return &sgtmpb.PostList_Response{Posts: posts}, nil
}
