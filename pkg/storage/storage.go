package storage

import (
	"errors"
	"fmt"

	"github.com/gosimple/slug"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"moul.io/sgtm/pkg/entities"
	"moul.io/sgtm/pkg/sgtmpb"
)

type Storage interface {
	GetMe(userID int64) (*sgtmpb.User, error)
	GetUsersList() ([]*sgtmpb.User, error)
	GetPostList(limit int) ([]*sgtmpb.Post, error)
	PatchUser(
		email string,
		userID string,
		avatar string,
		username string,
		locale string,
		discordID string,
		discriminator string,
	) (*sgtmpb.User, error)
	GetTrackByCID(cid string) (*sgtmpb.Post, error)
	GetTrackBySCID(scid uint64) (*sgtmpb.Post, error)
	GetUploadsByWeek() ([]*entities.UploadsByWeekDay, error)
	GetLastActivities(moulID int64) ([]*sgtmpb.Post, error)
	GetNumberOfDraftPosts() (int64, error)
	GetNumberOfUsers() (int64, error)
}

type storage struct {
	db *gorm.DB
}

func NewStorage(db *gorm.DB) Storage {
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

func (s *storage) GetPostList(limit int) ([]*sgtmpb.Post, error) {
	var posts []*sgtmpb.Post

	err := s.db.
		Order("sort_date desc").
		Where(sgtmpb.Post{
			Visibility: sgtmpb.Visibility_Public,
		}).
		Where("kind in (?)", sgtmpb.Post_TrackKind).
		Limit(limit).
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

func (s *storage) PatchUser(
	email string,
	userID string,
	avatar string,
	username string,
	locale string,
	discordID string,
	discriminator string,
) (*sgtmpb.User, error) {
	var dbUser sgtmpb.User
	{
		dbUser.Email = email
		err := s.db.Where(&dbUser).First(&dbUser).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			// user not found, creating it
			dbUser = sgtmpb.User{
				Email:           email,
				Avatar:          fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", userID, avatar),
				Slug:            slug.Make(username),
				Locale:          locale,
				DiscordID:       discordID,
				DiscordUsername: fmt.Sprintf("%s#%s", username, discriminator),
				// Firstname
				// Lastname
			}
			// FIXME: check if slug already exists, if yes, append something to the slug
			err = s.db.Omit(clause.Associations).Transaction(func(tx *gorm.DB) error {
				if err := tx.Create(&dbUser).Error; err != nil {
					return err
				}

				registerEvent := sgtmpb.Post{AuthorID: dbUser.ID, Kind: sgtmpb.Post_RegisterKind}
				if err := tx.Create(&registerEvent).Error; err != nil {
					return err
				}
				linkDiscordEvent := sgtmpb.Post{AuthorID: dbUser.ID, Kind: sgtmpb.Post_LinkDiscordAccountKind}
				if err := tx.Create(&linkDiscordEvent).Error; err != nil {
					return err
				}
				return nil
			})
			if err != nil {
				return nil, err
			}

		case err == nil:
			// user exists
			// FIXME: update user in DB if needed

			loginEvent := sgtmpb.Post{AuthorID: dbUser.ID, Kind: sgtmpb.Post_LoginKind}
			if err := s.db.Omit(clause.Associations).Create(&loginEvent).Error; err != nil {
				return nil, err
			}

		default:
			// unexpected error
			return nil, err
		}
	}
	return &dbUser, nil
}

func (s *storage) GetTrackByCID(cid string) (*sgtmpb.Post, error) {
	var post *sgtmpb.Post
	err := s.db.
		Model(&sgtmpb.Post{}).
		Where(sgtmpb.Post{IPFSCID: cid}).
		First(&post).
		Error
	if err != nil {
		return nil, err
	}
	return post, nil
}

func (s *storage) GetTrackBySCID(scid uint64) (*sgtmpb.Post, error) {
	var post *sgtmpb.Post
	err := s.db.
		Model(&sgtmpb.Post{}).
		Where(sgtmpb.Post{SoundCloudID: scid}).
		First(&post).
		Error
	if err != nil {
		return nil, err
	}
	return post, nil
}

func (s *storage) GetUploadsByWeek() ([]*entities.UploadsByWeekDay, error) {
	var upbyw []*entities.UploadsByWeekDay
	err := s.db.Model(&sgtmpb.Post{}).
		Where(&sgtmpb.Post{Kind: sgtmpb.Post_TrackKind}).
		Select(`strftime("%w", sort_date/1000000000, "unixepoch") as weekday , count(*) as quantity`).
		Group("weekday").Find(&upbyw).
		Error
	if err != nil {
		return nil, err
	}
	return upbyw, nil
}

func (s *storage) GetLastActivities(moulID int64) ([]*sgtmpb.Post, error) {
	var lastAct []*sgtmpb.Post
	err := s.db.
		Preload("Author").
		Preload("TargetPost").
		Preload("TargetUser").
		Order("created_at desc").
		Where("NOT (author_id == ? AND kind IN (?))", moulID, []sgtmpb.Post_Kind{
			sgtmpb.Post_ViewHomeKind,
			sgtmpb.Post_ViewOpenKind,
		}). // filter admin recurring actions
		// Where("author_id != 0"). // filter anonymous
		Where("kind NOT IN (?)", []sgtmpb.Post_Kind{
			sgtmpb.Post_LinkDiscordAccountKind,
			// sgtmpb.Post_LoginKind,
		}).
		Limit(42).
		Find(&lastAct).
		Error
	if err != nil {
		return nil, err
	}
	return lastAct, nil
}

func (s *storage) GetNumberOfDraftPosts() (int64, error) {
	var count int64
	err := s.db.
		Model(&sgtmpb.Post{}).
		Where(sgtmpb.Post{
			Kind:       sgtmpb.Post_TrackKind,
			Visibility: sgtmpb.Visibility_Draft,
		}).
		Count(&count).
		Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (s *storage) GetNumberOfUsers() (int64, error) {
	var count int64
	err := s.db.
		Model(&sgtmpb.User{}).
		Count(&count).
		Error
	if err != nil {
		return 0, err
	}
	return count, nil
}
