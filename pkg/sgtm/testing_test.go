package sgtm

import (
	"context"
	"flag"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"moul.io/sgtm/pkg/sgtmstore"
	"moul.io/zapconfig"
)

var debugFlag = flag.Bool("debug", false, "more verbose logging")

func TestingService(t *testing.T) (Service, func()) {
	opts := Opts{
		Logger:     TestingLogger(t),
		ServerBind: "127.0.0.1:0",
	}
	opts.applyDefaults()
	store := sgtmstore.TestingStore(t, opts.Logger)
	ctx, cancel := context.WithCancel(opts.Context)
	svc := Service{
		store:     store,
		logger:    opts.Logger,
		opts:      opts,
		ctx:       ctx,
		cancel:    cancel,
		StartedAt: time.Now(),
		ipfs:      ipfsWrapper{api: opts.IPFSAPI},
		unittest:  true,
	}
	gr, err := svc.prepareStartServer()
	require.NoError(t, err)
	go gr.Run()
	// wait for the server to be started
	time.Sleep(50 * time.Millisecond) // FIXME: replace by a better method
	cleanup := func() {
		svc.CloseServer(nil)
	}
	return svc, cleanup
}

func TestingLogger(t *testing.T) *zap.Logger {
	if *debugFlag {
		return zapconfig.Configurator{}.MustBuild()
	}
	return zap.NewNop()
}
