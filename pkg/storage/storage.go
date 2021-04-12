package storage

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/gosimple/slug"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"moul.io/sgtm/pkg/sgtmpb"
)

type Storage interface {
	GetUserByID(userID int64) (*sgtmpb.User, error)
	GetUsersList() ([]*sgtmpb.User, error)
	GetPostList(limit int) ([]*sgtmpb.Post, error)
	CreateUser(dbUser *sgtmpb.User) (*sgtmpb.User, error)
	GetTrackByCID(cid string) (*sgtmpb.Post, error)
	GetTrackBySCID(scid uint64) (*sgtmpb.Post, error)
	GetUploadsByWeek() ([]*sgtmpb.UploadsByWeek, error)
	GetLastActivities(moulID int64) ([]*sgtmpb.Post, error)
	GetNumberOfDraftPosts() (int64, error)
	GetNumberOfUsers() (int64, error)
	PatchPost(post *sgtmpb.Post) error
	GetNumberOfPostsByKind() ([]*sgtmpb.PostByKind, error)
	GetTotalDuration() (int64, error)
	GetPostBySugID(postSlug string) (*sgtmpb.Post, error)
	GetPostComments(postID int64) ([]*sgtmpb.Post, error)
	GetUserBySlug(slug string) (*sgtmpb.User, error)
	GetCalendarHeatMap(authorID int64) ([]int64, error)
	UpdatePost(post *sgtmpb.Post) error
	GenericUpdatePost(model interface{}, fields interface{}) error
	GetUserRecentPost(userID int64) (*sgtmpb.User, error)
	GetPostListByUserID(userID int64, limit int) ([]*sgtmpb.Post, error)
}

type storage struct {
	db *gorm.DB
}

func NewStorage(db *gorm.DB) Storage {
	return &storage{db: db}
}

func (s *storage) GetUserByID(userID int64) (*sgtmpb.User, error) {
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

func (s *storage) CreateUser(dbUser *sgtmpb.User) (*sgtmpb.User, error) {
	{
		err := s.db.Where(&dbUser).First(&dbUser).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			// user not found, creating it
			dbUser = &sgtmpb.User{
				Email:           dbUser.Email,
				Avatar:          fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", dbUser.DiscordID, dbUser.Avatar),
				Slug:            slug.Make(dbUser.Slug),
				Locale:          dbUser.Locale,
				DiscordID:       dbUser.DiscordID,
				DiscordUsername: fmt.Sprintf("%s#%s", dbUser.Slug, dbUser.DiscordUsername),
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
	return dbUser, nil
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

func (s *storage) GetUploadsByWeek() ([]*sgtmpb.UploadsByWeek, error) {
	var upbyw []*sgtmpb.UploadsByWeek
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

func (s *storage) PatchPost(post *sgtmpb.Post) error {
	return s.db.Omit(clause.Associations).Create(&post).Error
}

func (s *storage) GetNumberOfPostsByKind() ([]*sgtmpb.PostByKind, error) {
	var postsByKind []*sgtmpb.PostByKind
	err := s.db.
		Model(&sgtmpb.Post{}).
		// Where(sgtmpb.Post{Visibility: sgtmpb.Visibility_Public}).
		Select(`kind, count(*) as quantity`).
		Group("kind").
		Find(&postsByKind).
		Error
	if err != nil {
		return nil, err
	}
	return postsByKind, nil
}

func (s *storage) GetTotalDuration() (int64, error) {
	var totalDuration int64
	err := s.db.
		Model(&sgtmpb.Post{}).
		Select("sum(duration) as total_duration").
		Where(sgtmpb.Post{
			Kind: sgtmpb.Post_TrackKind,
			//Visibility: sgtmpb.Visibility_Public,
		}).
		First(&totalDuration).
		Error
	if err != nil {
		return 0, err
	}
	return totalDuration, nil
}

func (s *storage) GetPostBySugID(postSlug string) (*sgtmpb.Post, error) {
	query := s.db.
		Preload("Author").
		Preload("RelationshipsAsSource").
		Preload("RelationshipsAsSource.TargetPost").
		Preload("RelationshipsAsSource.TargetUser").
		Preload("RelationshipsAsTarget").
		Preload("RelationshipsAsTarget.SourcePost").
		Preload("RelationshipsAsTarget.SourceUser")
	id, err := strconv.ParseInt(postSlug, 10, 64)
	if err == nil {
		query = query.Where(sgtmpb.Post{ID: id, Kind: sgtmpb.Post_TrackKind})
	} else {
		query = query.Where(sgtmpb.Post{Slug: postSlug, Kind: sgtmpb.Post_TrackKind})
	}
	var post *sgtmpb.Post
	err = query.First(&post).Error
	if err != nil {
		return nil, err
	}
	return post, nil
}

func (s *storage) GetPostComments(postID int64) ([]*sgtmpb.Post, error) {
	var postComments []*sgtmpb.Post
	err := s.db.
		Where(sgtmpb.Post{
			Kind:         sgtmpb.Post_CommentKind,
			TargetPostID: postID,
			Visibility:   sgtmpb.Visibility_Public,
		}).
		Preload("Author").
		Find(&postComments).
		Error
	if err != nil {
		return nil, err
	}
	return postComments, nil
}

func (s *storage) GetUserBySlug(slug string) (*sgtmpb.User, error) {
	var user *sgtmpb.User
	err := s.db.
		Where("LOWER(slug) = ?", slug).
		First(&user).
		Error
	if err != nil {
		return nil, nil
	}
	return user, nil
}

func (s *storage) GetCalendarHeatMap(authorID int64) ([]int64, error) {
	var timestamps []int64
	err := s.db.Model(&sgtmpb.Post{}).
		Select(`sort_date/1000000000 as timestamp`).
		Where(sgtmpb.Post{
			AuthorID:   authorID,
			Kind:       sgtmpb.Post_TrackKind,
			Visibility: sgtmpb.Visibility_Public,
		}).
		Pluck("timestamp", &timestamps).
		Error
	if err != nil {
		return nil, err
	}
	return timestamps, nil
}

func (s *storage) UpdatePost(post *sgtmpb.Post) error {
	return s.db.Omit(clause.Associations).Where("id = ?", post.ID).Save(post).Error
}

func (s *storage) GenericUpdatePost(model interface{}, fields interface{}) error {
	return s.db.Omit(clause.Associations).Model(model).Updates(fields).Error
}

func (s *storage) GetUserRecentPost(userID int64) (*sgtmpb.User, error) {
	var user *sgtmpb.User
	err := s.db.
		Preload("RecentPosts", func(db *gorm.DB) *gorm.DB {
			return db.
				Where("kind IN (?)", []sgtmpb.Post_Kind{sgtmpb.Post_TrackKind}).
				Order("created_at desc").
				Limit(3)
		}).
		First(&user, userID).
		Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *storage) GetPostListByUserID(userID int64, limit int) ([]*sgtmpb.Post, error) {
	var tracks int64
	var posts []*sgtmpb.Post
	query := s.db.
		Model(&sgtmpb.Post{}).
		Where(sgtmpb.Post{
			AuthorID:   userID,
			Kind:       sgtmpb.Post_TrackKind,
			Visibility: sgtmpb.Visibility_Public,
		})
	err := query.Count(&tracks).Error
	if err != nil {
		return nil, err
	}
	if tracks > 0 {
		err := query.
			Order("sort_date desc").
			Limit(limit). // FIXME: pagination
			Find(&posts).
			Error
		if err != nil {
			return nil, err
		}
	}
	for _, track := range posts {
		track.ApplyDefaults()
	}
	return posts, nil
}
