package sgtm

import (
	"context"
	"flag"
	"testing"
	"time"

	"go.uber.org/zap"
	"moul.io/sgtm/pkg/sgtmstore"
	"moul.io/zapconfig"
)

var debugFlag = flag.Bool("debug", false, "more verbose logging")

func TestingService(t *testing.T) Service {
	opts := Opts{Logger: TestingLogger(t)}
	opts.applyDefaults()
	store := sgtmstore.TestingStore(t)
	ctx, cancel := context.WithCancel(opts.Context)
	svc := Service{
		store:     store,
		logger:    opts.Logger,
		opts:      opts,
		ctx:       ctx,
		cancel:    cancel,
		StartedAt: time.Now(),
	}
	return svc
}

func TestingLogger(t *testing.T) *zap.Logger {
	if *debugFlag {
		return zapconfig.Configurator{}.MustBuild()
	}
	return zap.NewNop()
}
