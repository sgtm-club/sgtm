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
			user, err := svc.storage.GetUserBySlug(userSlug)
			if err != nil {
				svc.error404Page(box)(w, r)
				return
			}
			data.Profile.User = user
		}

		// tracking
		{
			viewEvent := sgtmpb.Post{AuthorID: data.UserID, Kind: sgtmpb.Post_ViewProfileKind, TargetUserID: data.Profile.User.ID}
			err := svc.storage.PatchPost(&viewEvent)
			if err != nil {
				data.Error = "Cannot write activity: " + err.Error()
			} else {
				svc.logger.Debug("new view profile", zap.Any("event", &viewEvent))
			}
		}

		// tracks
		{
			data.Profile.LastTracks, err = svc.storage.GetPostList(100)
			if err != nil {
				data.Error = "Cannot fetch last tracks: " + err.Error()
			}
			for _, track := range data.Profile.LastTracks {
				track.ApplyDefaults()
			}
		}

		// calendar heatmap
		if data.Profile.Stats.Tracks > 0 {
			timestamps, err := svc.storage.GetCalendarHeatMap(data.Profile.User.ID)
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
