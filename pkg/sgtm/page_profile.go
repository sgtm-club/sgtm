package sgtm

import (
	"net/http"
	"time"

	"github.com/go-chi/chi"
	packr "github.com/gobuffalo/packr/v2"
	"go.uber.org/zap"
	"moul.io/sgtm/pkg/sgtmpb"
)

func (svc *Service) profilePage(box *packr.Box) func(w http.ResponseWriter, r *http.Request) {
	tmpl := loadTemplates(box, "base.tmpl.html", "profile.tmpl.html")
	return func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		data, err := svc.newTemplateData(w, r)
		if err != nil {
			svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
			return
		}
		// custom
		data.PageKind = "profile"

		// load profile
		{
			userSlug := chi.URLParam(r, "user_slug")
			var user sgtmpb.User
			if err := svc.rodb.
				Where(sgtmpb.User{Slug: userSlug}).
				First(&user).
				Error; err != nil {
				svc.error404Page(box)(w, r)
				return
			}
			data.Profile.User = &user
		}

		// tracking
		{
			viewEvent := sgtmpb.Post{AuthorID: data.UserID, Kind: sgtmpb.Post_ViewProfileKind, TargetUserID: data.Profile.User.ID}
			if err := svc.rwdb.Create(&viewEvent).Error; err != nil {
				data.Error = "Cannot write activity: " + err.Error()
			} else {
				svc.logger.Debug("new view profile", zap.Any("event", &viewEvent))
			}
		}

		// tracks
		{
			query := svc.rodb.
				Model(&sgtmpb.Post{}).
				Where(sgtmpb.Post{
					AuthorID:   data.Profile.User.ID,
					Kind:       sgtmpb.Post_TrackKind,
					Visibility: sgtmpb.Visibility_Public,
				})
			if err := query.Count(&data.Profile.Stats.Tracks).Error; err != nil {
				data.Error = "Cannot fetch last tracks: " + err.Error()
			}
			if data.Profile.Stats.Tracks > 0 {
				if err := query.
					Order("sort_date desc").
					Limit(50). // FIXME: pagination
					Find(&data.Profile.LastTracks).
					Error; err != nil {
					data.Error = "Cannot fetch last tracks: " + err.Error()
				}
			}
			for _, track := range data.Profile.LastTracks {
				track.ApplyDefaults()
			}
		}

		// calendar heatmap
		if data.Profile.Stats.Tracks > 0 {
			timestamps := []int64{}
			err := svc.rodb.Model(&sgtmpb.Post{}).
				Select(`sort_date/1000000000 as timestamp`).
				Where(sgtmpb.Post{
					AuthorID:   data.Profile.User.ID,
					Kind:       sgtmpb.Post_TrackKind,
					Visibility: sgtmpb.Visibility_Public,
				}).
				Pluck("timestamp", &timestamps).
				Error
			if err != nil {
				data.Error = "Cannot fetch post timestamps: " + err.Error()
			}
			data.Profile.CalendarHeatmap = map[int64]int64{}
			for _, timestamp := range timestamps {
				data.Profile.CalendarHeatmap[timestamp] = 1
			}
		}

		// end of custom
		if svc.opts.DevMode {
			tmpl = loadTemplates(box, "base.tmpl.html", "profile.tmpl.html")
		}
		data.Duration = time.Since(started)
		if err := tmpl.Execute(w, &data); err != nil {
			svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
			return
		}
	}
}
