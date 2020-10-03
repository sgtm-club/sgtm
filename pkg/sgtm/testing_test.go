package sgtm

import (
	"context"
	"flag"
	"testing"
	"time"

	"github.com/bwmarrin/snowflake"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"moul.io/zapconfig"
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
		_db:       db,
		logger:    opts.Logger,
		opts:      opts,
		ctx:       ctx,
		cancel:    cancel,
		StartedAt: time.Now(),
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

	db, err = DBInit(db, sfn)
	if err != nil {
		t.Fatalf("DBInit")
	}

	return db
}

func TestingLogger(t *testing.T) *zap.Logger {
	if *debugFlag {
		return zapconfig.Configurator{}.MustBuild()
	}
	return zap.NewNop()
}
