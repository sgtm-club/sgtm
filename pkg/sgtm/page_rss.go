package sgtm

import (
	"net/http"
	"time"

	packr "github.com/gobuffalo/packr/v2"
	"moul.io/sgtm/pkg/sgtmpb"
)

func (svc *Service) rssPage(box *packr.Box) func(w http.ResponseWriter, r *http.Request) {
	tmpl := loadTemplates(box, "rss.tmpl.xml")
	return func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		data, err := svc.newTemplateData(w, r)
		if err != nil {
			svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
			return
		}
		// custom
		w.Header().Add("Content-Type", "application/xml")
		// last tracks
		{
			if err := svc.db.
				Model(&sgtmpb.Post{}).
				Preload("Author").
				Where(sgtmpb.Post{
					Kind:       sgtmpb.Post_TrackKind,
					Visibility: sgtmpb.Visibility_Public,
				}).
				Order("sort_date desc").
				Limit(50). // FIXME: pagination
				Find(&data.RSS.LastTracks).
				Error; err != nil {
				svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
			}
			for _, track := range data.Home.LastTracks {
				track.ApplyDefaults()
			}
		}
		// end of custom
		if svc.opts.DevMode {
			tmpl = loadTemplates(box, "rss.tmpl.xml")
		}
		data.Duration = time.Since(started)
		if err := tmpl.ExecuteTemplate(w, "base", &data); err != nil {
			svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
			return
		}
	}
}
