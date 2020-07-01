package sgtm

import (
	"context"
	"flag"
	"testing"
	"time"

	"github.com/bwmarrin/snowflake"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"moul.io/zapgorm2"
)

var debugFlag = flag.Bool("debug", false, "more verbose logging")

func TestingService(t *testing.T) Service {
	db := TestingDB(t)
	opts := Opts{
		Logger: TestingLogger(t),
	}
	opts.applyDefaults()
	ctx, cancel := context.WithCancel(opts.Context)
	svc := Service{
		db:        db,
		logger:    opts.Logger,
		opts:      opts,
		ctx:       ctx,
		cancel:    cancel,
		startedAt: time.Now(),
	}
	return svc
}

func TestingDB(t *testing.T) *gorm.DB {
	t.Helper()

	logger := TestingLogger(t)
	zg := zapgorm2.New(logger)
	zg.LogLevel = gormlogger.Info
	zg.SetAsDefault()

	config := &gorm.Config{
		Logger:         zg,
		NamingStrategy: schema.NamingStrategy{TablePrefix: "sgtm_", SingularTable: true},
	}
	db, err := gorm.Open(sqlite.Open(":memory:"), config)
	if err != nil {
		t.Fatalf("gorm.Open")
	}

	sfn, err := snowflake.NewNode(1)
	if err != nil {
		t.Fatalf("snowflake.NewNode")
	}

	err = DBInit(db, sfn)
	if err != nil {
		t.Fatalf("DBInit")
	}

	return db
}

func TestingLogger(t *testing.T) *zap.Logger {
	if *debugFlag {
		config := zap.NewDevelopmentConfig()
		config.DisableStacktrace = true
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		config.Level.SetLevel(zap.DebugLevel)
		logger, err := config.Build()
		if err != nil {
			t.Errorf("setup debug logger error: `%v`", err)
			return zap.NewNop()
		}
		return logger
	}
	return zap.NewNop()
}
