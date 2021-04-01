package store

import (
	"fmt"

	"gorm.io/gorm"

	"moul.io/sgtm/pkg/sgtmpb"
)

type Store interface {
	GetUser(userID int64) (*sgtmpb.User, error)
}

type store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) Store {
	return &store{
		db: db,
	}
}

func (s *store) GetUser(userID int64) (*sgtmpb.User, error) {
	var user *sgtmpb.User
	err := s.db.
		Where("id = ?", userID).
		First(&user).
		Error
	if err != nil {
		return nil, fmt.Errorf("store: GetUser: %w", err)
	}
	return user, nil
}
