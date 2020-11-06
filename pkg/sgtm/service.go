package sgtm

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"moul.io/banner"
	"moul.io/sgtm/pkg/sgtmpb"
)

type Service struct {
	sgtmpb.UnimplementedWebAPIServer

	_db           *gorm.DB
	logger        *zap.Logger
	opts          Opts
	ctx           context.Context
	cancel        func()
	StartedAt     time.Time
	errRenderHTML func(w http.ResponseWriter, r *http.Request, err error, status int)

	// drivers

	discord discordDriver
	server  serverDriver
}

func New(db *gorm.DB, opts Opts) (Service, error) {
	if err := opts.applyDefaults(); err != nil {
		return Service{}, err
	}
	fmt.Fprintln(os.Stderr, banner.Inline("sgtm"))
	ctx, cancel := context.WithCancel(opts.Context)
	svc := Service{
		_db:       db,
		logger:    opts.Logger,
		opts:      opts,
		ctx:       ctx,
		cancel:    cancel,
		StartedAt: time.Now(),
	}
	svc.logger.Info("service initialized", zap.Bool("dev-mode", opts.DevMode))
	return svc, nil
}

func (svc *Service) Close() {
	svc.logger.Debug("closing service")
	svc.cancel()
	fmt.Fprintln(os.Stderr, banner.Inline("kthxbie"))
}
