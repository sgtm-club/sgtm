package sgtm

import (
	"github.com/bwmarrin/snowflake"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"moul.io/sgtm/pkg/sgtmpb"
)

func DBInit(db *gorm.DB, sfn *snowflake.Node, logger *zap.Logger) error {
	err := db.Callback().Create().Before("gorm:create").Register("sgtm_before_create", beforeCreate(sfn, logger))
	if err != nil {
		return err
	}

	err = db.AutoMigrate(
		&sgtmpb.User{},
		&sgtmpb.Post{},
	)
	if err != nil {
		return err
	}

	return nil
}

func beforeCreate(sfn *snowflake.Node, logger *zap.Logger) func(*gorm.DB) {
	return func(tx *gorm.DB) {
                tx.Statement.SetColumn("ID", sfn.Generate().Int64())
        }
}
