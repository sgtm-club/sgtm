package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"os"
	"syscall"

	"github.com/bwmarrin/snowflake"
	"github.com/oklog/run"
	ff "github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"moul.io/sgtm/pkg/sgtm"
	"moul.io/srand"
	"moul.io/zapgorm2"
)

func main() {
	err := app(os.Args)
	switch {
	case err == nil:
	case err == run.SignalError{Signal: os.Interrupt}:
	default:
		log.Fatalf("error: %v", err)
		os.Exit(1)
	}
}

var svcOpts sgtm.Opts

func app(args []string) error {
	svcOpts = sgtm.DefaultOpts()
	rootFlags := flag.NewFlagSet("root", flag.ExitOnError)
	rootFlags.BoolVar(&svcOpts.DevMode, "dev-mode", svcOpts.DevMode, "start in developer mode")
	/// discord
	rootFlags.BoolVar(&svcOpts.EnableDiscord, "enable-discord", svcOpts.EnableDiscord, "enable discord bot")
	rootFlags.StringVar(&svcOpts.DiscordToken, "discord-token", svcOpts.DiscordToken, "discord bot token")
	rootFlags.StringVar(&svcOpts.DiscordAdminChannel, "discord-admin-channel", svcOpts.DiscordAdminChannel, "discord channel ID for admin messages")
	/// server
	rootFlags.StringVar(&svcOpts.DBPath, "db-path", svcOpts.DBPath, "database path")
	rootFlags.BoolVar(&svcOpts.EnableServer, "enable-server", svcOpts.EnableServer, "enable HTTP+gRPC Server")
	rootFlags.StringVar(&svcOpts.ServerBind, "server-bind", svcOpts.ServerBind, "server bind (HTTP + gRPC)")
	rootFlags.StringVar(&svcOpts.ServerCORSAllowedOrigins, "server-cors-allowed-origins", svcOpts.ServerCORSAllowedOrigins, "allowed CORS origins")
	rootFlags.DurationVar(&svcOpts.ServerRequestTimeout, "server-request-timeout", svcOpts.ServerRequestTimeout, "server request timeout")
	rootFlags.DurationVar(&svcOpts.ServerShutdownTimeout, "server-shutdown-timeout", svcOpts.ServerShutdownTimeout, "server shutdown timeout")
	rootFlags.BoolVar(&svcOpts.ServerWithPprof, "server-with-pprof", svcOpts.ServerWithPprof, "enable pprof on HTTP server")
	rootFlags.StringVar(&svcOpts.DiscordClientID, "discord-client-id", svcOpts.DiscordClientID, "discord client ID (oauth)")
	rootFlags.StringVar(&svcOpts.DiscordClientSecret, "discord-client-secret", svcOpts.DiscordClientSecret, "discord client secret (oauth)")

	root := &ffcli.Command{
		FlagSet: rootFlags,
		Options: []ff.Option{
			ff.WithEnvVarPrefix("SGTM"),
			ff.WithConfigFile("config.txt"),
			ff.WithConfigFileParser(ff.PlainParser),
		},
		Subcommands: []*ffcli.Command{
			{Name: "run", Exec: runCmd},
		},
		Exec: func(context.Context, []string) error {
			return flag.ErrHelp
		},
	}

	return root.ParseAndRun(context.Background(), args[1:])
}

func runCmd(ctx context.Context, _ []string) error {
	// init
	rand.Seed(srand.Secure())

	// bearer
	// FIXME: TODO

	// zap logger
	{
		config := zap.NewDevelopmentConfig()
		config.Level.SetLevel(zap.DebugLevel)
		config.DisableStacktrace = true
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		logger, err := config.Build()
		if err != nil {
			return err
		}
		svcOpts.Logger = logger
	}

	// init db
	var db *gorm.DB
	{
		var err error
		zg := zapgorm2.New(svcOpts.Logger.Named("gorm"))
		zg.LogLevel = gormlogger.Info
		zg.SetAsDefault()
		config := &gorm.Config{
			Logger:         zg,
			NamingStrategy: schema.NamingStrategy{TablePrefix: "sgtm_", SingularTable: true},
		}
		db, err = gorm.Open(sqlite.Open(svcOpts.DBPath), config)
		if err != nil {
			return err
		}

		sfn, err := snowflake.NewNode(1)
		if err != nil {
			return err
		}
		err = sgtm.DBInit(db, sfn)
		if err != nil {
			return err
		}
	}

	// init service
	var svc sgtm.Service
	{
		//svcOpts.Context = ctx
		svc = sgtm.New(db, svcOpts)
		defer svc.Close()
	}

	// run.Group
	var gr run.Group
	{
		if svcOpts.EnableDiscord || svcOpts.EnableServer {
			gr.Add(run.SignalHandler(ctx, syscall.SIGTERM, syscall.SIGINT, os.Interrupt, os.Kill))
		}
		if svcOpts.EnableDiscord {
			gr.Add(svc.StartDiscord, svc.CloseDiscord)
		}
		if svcOpts.EnableServer {
			gr.Add(svc.StartServer, svc.CloseServer)
		}
	}
	return gr.Run()
}
