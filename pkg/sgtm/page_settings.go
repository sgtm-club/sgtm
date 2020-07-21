package sgtm

import (
	"net/http"
	"strings"
	"time"

	packr "github.com/gobuffalo/packr/v2"
	"go.uber.org/zap"
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
			validate := func() map[string]interface{} {
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

				fields := map[string]interface{}{
					"firstname":           strings.TrimSpace(r.Form.Get("firstname")),
					"lastname":            strings.TrimSpace(r.Form.Get("lastname")),
					"homepage":            strings.TrimSpace(r.Form.Get("homepage")),
					"bio":                 strings.TrimSpace(r.Form.Get("bio")),
					"headline":            strings.TrimSpace(r.Form.Get("headline")),
					"inspirations":        strings.TrimSpace(r.Form.Get("inspirations")),
					"gears":               strings.TrimSpace(r.Form.Get("gears")),
					"goals":               strings.TrimSpace(r.Form.Get("goals")),
					"genres":              strings.TrimSpace(r.Form.Get("genres")),
					"other_links":         strings.TrimSpace(r.Form.Get("other_links")),
					"twitter_username":    twitter,
					"soundcloud_username": soundcloud,
				}
				return fields
			}
			fields := validate()
			if fields != nil {
				if err := svc.db.Model(data.User).Omit("RecentPosts").Updates(fields).Error; err != nil {
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
