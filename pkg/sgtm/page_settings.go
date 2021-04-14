package sgtm

import (
	"net/http"
	"strings"
	"time"

	packr "github.com/gobuffalo/packr/v2"
	"go.uber.org/zap"

	"moul.io/sgtm/pkg/sgtmpb"
)

func (svc *Service) settingsPage(box *packr.Box) func(w http.ResponseWriter, r *http.Request) {
	tmpl := loadTemplates(box, "base.tmpl.html", "settings.tmpl.html")
	return func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		data, err := svc.newTemplateData(w, r)
		if err != nil {
			svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
			return
		}
		// custom
		data.PageKind = "settings"
		if data.User == nil {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}
		if r.Method == "POST" {
			validate := func() *sgtmpb.User {
				if err := r.ParseForm(); err != nil {
					data.Error = err.Error()
					return nil
				}
				// FIXME: blacklist, etc
				twitter := strings.TrimSpace(r.Form.Get("twitter_username"))
				twitter = strings.TrimPrefix(twitter, "https://twitter.com/")
				twitter = strings.TrimPrefix(twitter, "@")
				soundcloud := strings.TrimSpace(r.Form.Get("soundcloud_username"))
				soundcloud = strings.TrimPrefix(soundcloud, "https://soundcloud.com/")
				soundcloud = strings.TrimPrefix(soundcloud, "@")

				user := &sgtmpb.User{
					Firstname:          strings.TrimSpace(r.Form.Get("firstname")),
					Lastname:           strings.TrimSpace(r.Form.Get("lastname")),
					Homepage:           strings.TrimSpace(r.Form.Get("homepage")),
					Bio:                strings.TrimSpace(r.Form.Get("bio")),
					Headline:           strings.TrimSpace(r.Form.Get("headline")),
					Inspirations:       strings.TrimSpace(r.Form.Get("inspirations")),
					Gears:              strings.TrimSpace(r.Form.Get("gears")),
					Goals:              strings.TrimSpace(r.Form.Get("goals")),
					Genres:             strings.TrimSpace(r.Form.Get("genres")),
					OtherLinks:         strings.TrimSpace(r.Form.Get("other_links")),
					TwitterUsername:    twitter,
					SoundcloudUsername: soundcloud,
				}

				return user
			}
			fields := validate()
			if fields != nil {
				err := svc.storage.UpdateUser(fields, fields)
				if err != nil {
					svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
					return
				}
				svc.logger.Debug("settings update", zap.Any("fields", fields))
				http.Redirect(w, r, "/settings", http.StatusFound)
				return
			}
		}
		// end of custom
		if svc.opts.DevMode {
			tmpl = loadTemplates(box, "base.tmpl.html", "settings.tmpl.html")
		}
		data.Duration = time.Since(started)
		if err := tmpl.Execute(w, &data); err != nil {
			svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
			return
		}
	}
}
