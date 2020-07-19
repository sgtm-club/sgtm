package sgtm

import (
	"net/http"
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
				fields := sgtmpb.User{
					Firstname: r.Form.Get("firstname"),
					Lastname:  r.Form.Get("lastname"),
				}
				return &fields
			}
			fields := validate()
			if fields != nil {
				if err := svc.db.Model(data.User).Updates(fields).Error; err != nil {
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
