package sgtm

import (
	"net/http"
	"time"

	"github.com/gobuffalo/packr/v2"
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
			data.RSS.LastTracks, err = svc.storage.GetPostList(50)
			if err != nil {
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
