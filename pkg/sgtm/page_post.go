package sgtm

import (
	"net/http"
	"strconv"
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
		postSlug := chi.URLParam(r, "post_slug")
		query := svc.db.Preload("Author")
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
			if err := svc.db.Create(&viewEvent).Error; err != nil {
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
		query := svc.db.Preload("Author")
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

		http.Redirect(w, r, post.CanonicalURL(), http.StatusFound)
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
		if data.User == nil {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}
		postSlug := chi.URLParam(r, "post_slug")
		query := svc.db.Preload("Author")
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
		if !data.IsAdmin && data.User.ID != post.Author.ID {
			svc.error404Page(box)(w, r)
			return
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
