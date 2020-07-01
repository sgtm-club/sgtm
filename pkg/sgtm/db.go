package sgtm

import (
	"github.com/bwmarrin/snowflake"
	"gorm.io/gorm"
	"moul.io/sgtm/pkg/sgtmpb"
)

func DBInit(db *gorm.DB, sfn *snowflake.Node) error {
	err := db.Callback().Create().Before("gorm:create").Register("sgtm_before_create", beforeCreate(sfn))
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

func beforeCreate(sfn *snowflake.Node) func(*gorm.DB) {
	return func(tx *gorm.DB) {
		tx.Statement.SetColumn("ID", sfn.Generate().Int64())
	}
}
