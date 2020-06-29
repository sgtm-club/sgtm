package sgtm

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"moul.io/godev"
	"moul.io/sgtm/pkg/sgtmpb"
)

func TestServiceRegister(t *testing.T) {
	svc := TestingService(t)
	logger := TestingLogger(t)
	ctx := context.Background()

	tests := []struct {
		name            string
		input           *sgtmpb.Register_Request
		expectedOutput  *sgtmpb.Register_Response
		shouldHaveError bool
	}{
		{"nil", nil, nil, true},
		{"empty", &sgtmpb.Register_Request{}, nil, true},
		{"no-email", &sgtmpb.Register_Request{Slug: "moul", Firstname: "Manfred", Lastname: "Touron"}, nil, true},
		{"no-slug", &sgtmpb.Register_Request{Email: "m@42.am", Firstname: "Manfred", Lastname: "Touron"}, nil, true},
		{
			"manfred",
			&sgtmpb.Register_Request{Slug: "moul", Firstname: "Manfred", Lastname: "Touron", Email: "m@42.am"},
			&sgtmpb.Register_Response{User: &sgtmpb.User{Slug: "moul", Firstname: "Manfred", Lastname: "Touron", Email: "m@42.am"}},
			false,
		},
		{
			"manfred.bis",
			&sgtmpb.Register_Request{Slug: "moul", Firstname: "Manfred", Lastname: "Touron", Email: "m@42.am"},
			nil,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ret, err := svc.Register(ctx, tt.input)
			if tt.shouldHaveError {
				logger.Debug("err", zap.Error(err))
				require.Error(t, err)
				require.Nil(t, ret)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, ret)
			require.NotNil(t, ret.User)
			require.NotZero(t, ret.User.ID)
			require.NotZero(t, ret.User.CreatedAt)
			require.NotZero(t, ret.User.UpdatedAt)
			fmt.Println(godev.PrettyJSON(ret))
		})
	}
}
