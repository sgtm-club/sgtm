package sgtmstore

import (
	"testing"

	"github.com/bwmarrin/snowflake"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"moul.io/zapgorm2"
)

func TestingStore(t *testing.T, logger *zap.Logger) Store {
	t.Helper()

	zg := zapgorm2.New(logger)
	zg.LogLevel = gormlogger.Info
	zg.SetAsDefault()

	config := &gorm.Config{
		Logger: zg,
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "sgtm_",
			SingularTable: true,
		},
	}
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), config)
	require.NoError(t, err)

	sfn, err := snowflake.NewNode(1)
	require.NoError(t, err)

	store, err := New(db, sfn)
	require.NoError(t, err)

	return store
}
