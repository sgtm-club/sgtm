package sgtm

import (
	"reflect"

	"github.com/bwmarrin/snowflake"
	"gorm.io/gorm"
	"moul.io/sgtm/pkg/sgtmpb"
)

func DBInit(db *gorm.DB, sfn *snowflake.Node) error {
	db.Callback().Create().Before("gorm:create").Register("sgtm_before_create", beforeCreate(sfn))

	err := db.AutoMigrate(
		&sgtmpb.User{},
		&sgtmpb.Post{},
	)
	if err != nil {
		return err
	}

	return nil
}

func beforeCreate(sfn *snowflake.Node) func(*gorm.DB) {
	return func(db *gorm.DB) {
		s := reflect.ValueOf(db.Statement.Dest).Elem()
		if s.Kind() == reflect.Struct {
			f := s.FieldByName("ID")
			if f.IsValid() {
				if f.CanSet() {
					id := sfn.Generate().Int64()
					f.SetInt(id)
					return
				}
			}
		}
		panic("SOMETHING WRONG HAPPENED")
	}
}
