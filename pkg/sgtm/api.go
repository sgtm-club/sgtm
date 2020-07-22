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

	var user sgtmpb.User
	err = svc.rodb.
		Where("id = ?", claims.Session.UserID).
		First(&user).
		Error
	if err != nil {
		return nil, err
	}
	ret := sgtmpb.Me_Response{User: &user}

	return &ret, nil
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
	ret := &sgtmpb.UserList_Response{}
	err := svc.rodb.
		Order("created_at desc").
		Find(&ret.Users).
		Error
	if err != nil {
		return nil, err
	}

	for _, user := range ret.Users {
		user.Filter()
	}
	return ret, nil
}

func (svc *Service) PostList(context.Context, *sgtmpb.PostList_Request) (*sgtmpb.PostList_Response, error) {
	ret := &sgtmpb.PostList_Response{}
	err := svc.rodb.
		Order("sort_date desc").
		Where(sgtmpb.Post{
			Visibility: sgtmpb.Visibility_Public,
		}).
		Where("kind in (?)", sgtmpb.Post_TrackKind).
		Limit(100).
		Find(&ret.Posts).
		Error
	if err != nil {
		return nil, err
	}

	for _, post := range ret.Posts {
		post.Filter()
	}

	return ret, nil
}
