package sgtm

import (
	"gorm.io/gorm"

	"moul.io/sgtm/pkg/sgtmpb"
)

type Storage interface {
	GetMe(userID int64) (*sgtmpb.User, error)
	GetUsersList() ([]*sgtmpb.User, error)
	GetPostList() ([]*sgtmpb.Post, error)
}

type storage struct {
	db *gorm.DB
}

func NewStorage(db *gorm.DB) *storage {
	return &storage{db: db}
}

func (s *storage) GetMe(userID int64) (*sgtmpb.User, error) {
	var user *sgtmpb.User

	err := s.db.
		Where("id = ?", userID).
		First(&user).
		Error
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *storage) GetUsersList() ([]*sgtmpb.User, error) {
	var users []*sgtmpb.User
	err := s.db.
		Order("created_at desc").
		Find(&users).
		Error
	if err != nil {
		return nil, err
	}

	for _, u := range users {
		u.Filter()
	}
	return users, nil
}

func (s *storage) GetPostList() ([]*sgtmpb.Post, error) {
	var posts []*sgtmpb.Post

	err := s.db.
		Order("sort_date desc").
		Where(sgtmpb.Post{
			Visibility: sgtmpb.Visibility_Public,
		}).
		Where("kind in (?)", sgtmpb.Post_TrackKind).
		Limit(100).
		Find(&posts).
		Error
	if err != nil {
		return nil, err
	}

	for _, post := range posts {
		post.Filter()
	}

	return posts, nil
}
