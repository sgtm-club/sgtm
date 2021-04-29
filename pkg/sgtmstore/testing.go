package sgtmstore

import (
	"testing"

	"github.com/bwmarrin/snowflake"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"moul.io/zapconfig"
	"moul.io/zapgorm2"
)

func TestingStore(t *testing.T) Store {
	t.Helper()

	logger := zapconfig.Configurator{}.MustBuild()
	zg := zapgorm2.New(logger)
	zg.LogLevel = gormlogger.Info
	zg.SetAsDefault()

	config := &gorm.Config{
		Logger:         zg,
		NamingStrategy: schema.NamingStrategy{TablePrefix: "sgtm_", SingularTable: true},
	}
	db, err := gorm.Open(sqlite.Open(":memory:"), config)
	require.NoError(t, err)

	sfn, err := snowflake.NewNode(1)
	require.NoError(t, err)

	store, err := New(db, sfn)
	require.NoError(t, err)

	return store
}