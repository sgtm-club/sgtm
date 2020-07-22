package sgtm

import (
	"net/http"
	"time"

	packr "github.com/gobuffalo/packr/v2"
	"go.uber.org/zap"
	"moul.io/sgtm/pkg/sgtmpb"
)

func (svc *Service) homePage(box *packr.Box) func(w http.ResponseWriter, r *http.Request) {
	tmpl := loadTemplates(box, "base.tmpl.html", "home.tmpl.html")
	return func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		data, err := svc.newTemplateData(w, r)
		if err != nil {
			svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
			return
		}
		// custom
		data.PageKind = "home"

		// tracking
		{
			viewEvent := sgtmpb.Post{AuthorID: data.UserID, Kind: sgtmpb.Post_ViewHomeKind}
			if err := svc.rwdb.Create(&viewEvent).Error; err != nil {
				data.Error = "Cannot write activity: " + err.Error()
			} else {
				svc.logger.Debug("new view home", zap.Any("event", &viewEvent))
			}
		}

		// last tracks
		{
			limit := 50
			if data.UserID == 0 {
				limit = 10
			}
			if err := svc.rodb.
				Model(&sgtmpb.Post{}).
				Preload("Author").
				Where(sgtmpb.Post{
					Kind:       sgtmpb.Post_TrackKind,
					Visibility: sgtmpb.Visibility_Public,
				}).
				Order("sort_date desc").
				Limit(limit). // FIXME: pagination
				Find(&data.Home.LastTracks).
				Error; err != nil {
				data.Error = "Cannot fetch last tracks: " + err.Error()
			}
			for _, track := range data.Home.LastTracks {
				track.ApplyDefaults()
			}
		}

		// last users
		{
			if err := svc.rodb.
				Model(&sgtmpb.User{}).
				Order("created_at desc").
				Limit(10).
				Find(&data.Home.LastUsers).
				Error; err != nil {
				data.Error = "Cannot fetch last users: " + err.Error() // FIXME: use slice instead of string
			}
			for _, user := range data.Home.LastUsers {
				user.ApplyDefaults()
			}
		}
		// end of custom
		if svc.opts.DevMode {
			tmpl = loadTemplates(box, "base.tmpl.html", "home.tmpl.html")
		}
		data.Duration = time.Since(started)
		if err := tmpl.ExecuteTemplate(w, "base", &data); err != nil {
			svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
			return
		}
	}
}
