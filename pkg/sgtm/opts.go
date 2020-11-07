package sgtm

import (
	"context"
	"time"

	"github.com/bwmarrin/snowflake"
	"go.uber.org/zap"
)

const filteringString = "*FILTERED*"

type Opts struct {
	Context       context.Context
	Logger        *zap.Logger
	DevMode       bool
	JWTSigningKey string
	Snowflake     *snowflake.Node
	BearerToken   string

	// Discord

	EnableDiscord       bool
	DiscordToken        string
	DiscordAdminChannel string
	DiscordClientID     string
	DiscordClientSecret string

	// SoundCloud

	SoundCloudClientID string

	// DB

	DBPath string

	// Server

	EnableServer             bool
	ServerBind               string
	ServerCORSAllowedOrigins string
	ServerRequestTimeout     time.Duration
	ServerShutdownTimeout    time.Duration
	ServerWithPprof          bool
	Hostname                 string

	// IPFS

	IPFSAPI string // multiaddress or empty string to use the cli without "--api" option
}

func (opts *Opts) applyDefaults() error {
	if opts.Context == nil {
		opts.Context = context.TODO()
	}
	if opts.Logger == nil {
		opts.Logger = zap.NewNop()
	}
	if opts.Snowflake == nil {
		var err error
		opts.Snowflake, err = snowflake.NewNode(1)
		if err != nil {
			return err
		}
	}
	if opts.JWTSigningKey == "" {
		opts.JWTSigningKey = randString(42)
	}
	return nil
}

func (opts *Opts) setDefaults() {
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

func (opts Opts) Filtered() Opts {
	filtered := opts
	if filtered.DiscordToken != "" {
		filtered.DiscordToken = filteringString
	}
	if filtered.DiscordAdminChannel != "" {
		filtered.DiscordAdminChannel = filteringString
	}
	if filtered.DiscordClientSecret != "" {
		filtered.DiscordClientSecret = filteringString
	}
	if filtered.JWTSigningKey != "" {
		filtered.JWTSigningKey = filteringString
	}
	return filtered
}

func DefaultOpts() Opts {
	opts := Opts{}
	opts.setDefaults()
	return opts
}
