package sgtm

import (
	"github.com/bwmarrin/snowflake"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"moul.io/sgtm/pkg/sgtmpb"
)

func DBInit(db *gorm.DB, sfn *snowflake.Node) (*gorm.DB, error) {
	err := db.Callback().Create().Before("gorm:create").Register("sgtm_before_create", beforeCreate(sfn))
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(
		&sgtmpb.User{},
		&sgtmpb.Post{},
		&sgtmpb.Relationship{},
	)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func beforeCreate(sfn *snowflake.Node) func(*gorm.DB) {
	return func(tx *gorm.DB) {
		tx.Statement.SetColumn("ID", sfn.Generate().Int64())
	}
}

func (svc *Service) rodb() *gorm.DB { return svc._db }
func (svc *Service) rwdb() *gorm.DB { return svc._db.Omit(clause.Associations) }
