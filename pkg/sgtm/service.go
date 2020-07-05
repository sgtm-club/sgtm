package sgtm

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"moul.io/banner"
)

type Service struct {
	db        *gorm.DB
	logger    *zap.Logger
	opts      Opts
	ctx       context.Context
	cancel    func()
	startedAt time.Time

	/// drivers

	discord discordDriver
	server  serverDriver
}

func New(db *gorm.DB, opts Opts) Service {
	opts.applyDefaults()
	fmt.Fprintln(os.Stderr, banner.Inline("sgtm"))
	ctx, cancel := context.WithCancel(opts.Context)
	svc := Service{
		db:        db,
		logger:    opts.Logger,
		opts:      opts,
		ctx:       ctx,
		cancel:    cancel,
		startedAt: time.Now(),
	}
	svc.logger.Info("service initialized", zap.Bool("dev-mode", opts.DevMode))
	return svc
}

func (svc *Service) Close() {
	svc.logger.Debug("closing service")
	svc.cancel()
	fmt.Fprintln(os.Stderr, banner.Inline("kthxbie"))
}
