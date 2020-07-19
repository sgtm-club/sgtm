package sgtm

import (
	"net/http"
	"time"

	packr "github.com/gobuffalo/packr/v2"
)

func (svc *Service) error404Page(box *packr.Box) func(w http.ResponseWriter, r *http.Request) {
	tmpl := loadTemplates(box, "base.tmpl.html", "error404.tmpl.html")
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)

		started := time.Now()
		data, err := svc.newTemplateData(r)
		if err != nil {
			svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
			return
		}
		// custom
		// end of custom
		if svc.opts.DevMode {
			tmpl = loadTemplates(box, "base.tmpl.html", "error404.tmpl.html")
		}
		data.Duration = time.Since(started)
		if err := tmpl.Execute(w, &data); err != nil {
			svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
			return
		}
	}
}

func (svc *Service) errorPage(box *packr.Box) func(w http.ResponseWriter, r *http.Request, err error, status int) {
	tmpl := loadTemplates(box, "base.tmpl.html", "error.tmpl.html")
	return func(w http.ResponseWriter, r *http.Request, userError error, status int) {
		started := time.Now()
		data, err := svc.newTemplateData(r)
		if err != nil {
			svc.errRender(w, r, err, http.StatusUnprocessableEntity)
			return
		}
		// custom
		w.WriteHeader(status)
		if userError != nil {
			data.Error = userError.Error()
		}
		// end of custom
		if svc.opts.DevMode {
			tmpl = loadTemplates(box, "base.tmpl.html", "error.tmpl.html")
		}
		data.Duration = time.Since(started)
		if err := tmpl.Execute(w, &data); err != nil {
			svc.errRender(w, r, err, http.StatusUnprocessableEntity)
			return
		}
	}
}
