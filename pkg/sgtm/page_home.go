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
			err = svc.storage.PatchPost(&viewEvent)
			if err != nil {
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
			data.Home.LastTracks, err = svc.storage.GetPostList(limit)
			if err != nil {
				data.Error = "Cannot fetch last tracks: " + err.Error()
			}
			for _, track := range data.Home.LastTracks {
				track.ApplyDefaults()
			}
		}

		// last users
		{
			if data.Home.LastUsers, err = svc.storage.GetUsersList(); err != nil {
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
