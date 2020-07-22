package sgtm

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi"
	packr "github.com/gobuffalo/packr/v2"
	"go.uber.org/zap"
	"moul.io/sgtm/pkg/sgtmpb"
)

func (svc *Service) postPage(box *packr.Box) func(w http.ResponseWriter, r *http.Request) {
	tmpl := loadTemplates(box, "base.tmpl.html", "post.tmpl.html")
	return func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		data, err := svc.newTemplateData(w, r)
		if err != nil {
			svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
			return
		}
		// custom
		data.PageKind = "post"
		postSlug := chi.URLParam(r, "post_slug")
		query := svc.rodb.Preload("Author")
		id, err := strconv.ParseInt(postSlug, 10, 64)
		if err == nil {
			query = query.Where(sgtmpb.Post{ID: id, Kind: sgtmpb.Post_TrackKind})
		} else {
			query = query.Where(sgtmpb.Post{Slug: postSlug, Kind: sgtmpb.Post_TrackKind})
		}
		var post sgtmpb.Post
		if err := query.First(&post).Error; err != nil {
			svc.error404Page(box)(w, r)
			return
		}
		data.Post.Post = &post
		data.Post.Post.ApplyDefaults()

		// tracking
		{
			viewEvent := sgtmpb.Post{AuthorID: data.UserID, Kind: sgtmpb.Post_ViewPostKind, TargetPostID: data.Post.Post.ID}
			if err := svc.rwdb.Create(&viewEvent).Error; err != nil {
				data.Error = "Cannot write activity: " + err.Error()
			} else {
				svc.logger.Debug("new view post", zap.Any("event", &viewEvent))
			}
		}

		// end of custom
		if svc.opts.DevMode {
			tmpl = loadTemplates(box, "base.tmpl.html", "post.tmpl.html")
		}
		data.Duration = time.Since(started)
		if err := tmpl.Execute(w, &data); err != nil {
			svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
			return
		}
	}
}

func (svc *Service) postSyncPage(box *packr.Box) func(w http.ResponseWriter, r *http.Request) {
	tmpl := loadTemplates(box, "base.tmpl.html", "dummy.tmpl.html")
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
		postSlug := chi.URLParam(r, "post_slug")
		query := svc.rodb.Preload("Author")
		id, err := strconv.ParseInt(postSlug, 10, 64)
		if err == nil {
			query = query.Where(sgtmpb.Post{ID: id, Kind: sgtmpb.Post_TrackKind})
		} else {
			query = query.Where(sgtmpb.Post{Slug: postSlug, Kind: sgtmpb.Post_TrackKind})
		}
		var post sgtmpb.Post
		if err := query.First(&post).Error; err != nil {
			svc.error404Page(box)(w, r)
			return
		}
		if !data.IsAdmin && data.User.ID != post.Author.ID {
			svc.error404Page(box)(w, r)
			return
		}

		// FIXME: do the sync here
		dl, err := DownloadPost(&post, false)
		if err != nil {
			svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
			return
		}
		svc.logger.Debug("file downloaded", zap.String("path", dl.Path))
		bpm, err := ExtractBPM(dl.Path)
		if err != nil {
			svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
			return
		}
		svc.logger.Debug("BPM extracted", zap.Float64("bpm", bpm))
		if err := svc.rwdb.Model(&post).Update("bpm", bpm).Error; err != nil {
			svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
			return
		}

		switch r.URL.Query().Get("return") {
		case "no":
			return
		case "edit":
			http.Redirect(w, r, post.CanonicalURL()+"/edit", http.StatusFound)
		default:
			http.Redirect(w, r, post.CanonicalURL(), http.StatusFound)
		}
		// end of custom
		if svc.opts.DevMode {
			tmpl = loadTemplates(box, "base.tmpl.html", "dummy.tmpl.html")
		}
		data.Duration = time.Since(started)
		if err := tmpl.Execute(w, &data); err != nil {
			svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
			return
		}
	}
}

func (svc *Service) postEditPage(box *packr.Box) func(w http.ResponseWriter, r *http.Request) {
	tmpl := loadTemplates(box, "base.tmpl.html", "post-edit.tmpl.html")
	return func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		data, err := svc.newTemplateData(w, r)
		if err != nil {
			svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
			return
		}
		// custom
		data.PageKind = "post-edit"

		// no anonymous users
		{
			if data.User == nil {
				http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
				return
			}
		}

		// fetch post from db
		{
			postSlug := chi.URLParam(r, "post_slug")
			query := svc.rodb.Preload("Author")
			id, err := strconv.ParseInt(postSlug, 10, 64)
			if err == nil {
				query = query.Where(sgtmpb.Post{ID: id, Kind: sgtmpb.Post_TrackKind})
			} else {
				query = query.Where(sgtmpb.Post{Slug: postSlug, Kind: sgtmpb.Post_TrackKind})
			}
			var post sgtmpb.Post
			if err := query.First(&post).Error; err != nil {
				svc.error404Page(box)(w, r)
				return
			}
			data.PostEdit.Post = &post
		}

		// only author or admin
		{
			if !data.IsAdmin && data.User.ID != data.PostEdit.Post.Author.ID {
				svc.error404Page(box)(w, r)
				return
			}
		}

		// if POST
		if r.Method == "POST" {
			validate := func() map[string]interface{} {
				if err := r.ParseForm(); err != nil {
					data.Error = err.Error()
					return nil
				}
				// FIXME: blacklist, etc
				fields := map[string]interface{}{}
				fields["body"] = strings.TrimSpace(r.Form.Get("body"))
				return fields
			}
			fields := validate()
			if fields != nil {
				if err := svc.rwdb.Model(data.PostEdit.Post).Updates(fields).Error; err != nil {
					svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
					return
				}
				svc.logger.Debug("post updated", zap.Any("fields", fields))
				http.Redirect(w, r, data.PostEdit.Post.CanonicalURL(), http.StatusFound)
				return
			}
		}

		// end of custom
		if svc.opts.DevMode {
			tmpl = loadTemplates(box, "base.tmpl.html", "post-edit.tmpl.html")
		}
		data.Duration = time.Since(started)
		if err := tmpl.Execute(w, &data); err != nil {
			svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
			return
		}
	}
}
