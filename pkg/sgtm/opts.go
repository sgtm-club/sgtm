package sgtm

import (
	"context"
	"time"

	"go.uber.org/zap"
)

type Opts struct {
	Context context.Context
	Logger  *zap.Logger
	DevMode bool
	Seed    string

	/// Discord

	EnableDiscord       bool
	DiscordToken        string
	DiscordAdminChannel string
	DiscordClientID     string
	DiscordClientSecret string

	/// DB

	DBPath string

	/// Server

	EnableServer             bool
	ServerBind               string
	ServerCORSAllowedOrigins string
	ServerRequestTimeout     time.Duration
	ServerShutdownTimeout    time.Duration
	ServerWithPprof          bool
}

func (opts *Opts) applyDefaults() {
	if opts.Context == nil {
		opts.Context = context.TODO()
	}
	if opts.Logger == nil {
		opts.Logger = zap.NewNop()
	}
	if opts.ServerBind == "" {
		opts.ServerBind = ":8000"
	}
	if opts.ServerCORSAllowedOrigins == "" {
		opts.ServerCORSAllowedOrigins = "*"
	}
	if opts.ServerRequestTimeout == 0 {
		opts.ServerRequestTimeout = 5 * time.Second
	}
	if opts.ServerShutdownTimeout == 0 {
		opts.ServerShutdownTimeout = 6 * time.Second
	}
	if opts.DBPath == "" {
		opts.DBPath = "/tmp/sgtm.db"
	}
}

func (opts *Opts) Filtered() Opts {
	filtered := *opts
	if filtered.DiscordToken != "" {
		filtered.DiscordToken = "*FILTERED*"
	}
	if filtered.DiscordAdminChannel != "" {
		filtered.DiscordAdminChannel = "*FILTERED*"
	}
	return filtered
}

func DefaultOpts() Opts {
	opts := Opts{}
	opts.applyDefaults()
	return opts
}
