package sgtm

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/gobuffalo/packr/v2"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"moul.io/godev"

	"moul.io/sgtm/pkg/sgtmpb"
)

var (
	featRegex = regexp.MustCompile(`(?im)(feat.|feat|featuring|features)\s*[:= ]\s*@([^\s,]+)\s*`)
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
		post, err := svc.storage.GetCustomPost(postSlug)
		if err != nil {
			svc.error404Page(box)(w, r)
			return
		}
		data.Post.Post = post
		data.Post.Post.ApplyDefaults()

		if r.URL.Query().Get("format") == "json" {
			data.Post.Post.Filter()
			data.Post.Post.Author.Filter()
			fmt.Fprintln(w, godev.PrettyJSONPB(data.Post.Post))
			return
		}

		if r.Method == "POST" && data.UserID != 0 {
			validate := func() *sgtmpb.Post {
				if err := r.ParseForm(); err != nil {
					data.Error = err.Error()
					return nil
				}
				comment := sgtmpb.Post{
					Kind:         sgtmpb.Post_CommentKind,
					AuthorID:     data.UserID,
					Body:         strings.TrimSpace(r.Form.Get("comment")),
					Visibility:   sgtmpb.Visibility_Public,
					TargetPostID: data.Post.Post.ID,
				}
				if comment.Body == "" {
					return nil
				}
				return &comment
			}
			comment := validate()
			if comment != nil {
				err = svc.storage.PatchPost(comment)
				if err != nil {
					svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
					return
				}
				svc.logger.Debug("comment created", zap.Any("post", comment))
				http.Redirect(w, r, data.Post.Post.CanonicalURL(), http.StatusFound)
				return
			}
		}

		// load comments
		{
			data.Post.Comments, err = svc.storage.GetPostComments(data.Post.Post.ID)
			if err != nil {
				svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
				return
			}
		}

		// tracking
		{
			viewEvent := sgtmpb.Post{AuthorID: data.UserID, Kind: sgtmpb.Post_ViewPostKind, TargetPostID: data.Post.Post.ID}
			if err := svc.storage.PatchPost(&viewEvent); err != nil {
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

func (svc *Service) postMaintenancePage(box *packr.Box) func(w http.ResponseWriter, r *http.Request) {
	tmpl := loadTemplates(box, "base.tmpl.html", "dummy.tmpl.html")
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			shouldExtractBpm          = r.URL.Query().Get("extract_bpm") == "1"
			shouldDetectRelationships = r.URL.Query().Get("detect_relationships") == "1"
			shouldResyncSoundCloud    = r.URL.Query().Get("resync_soundcloud") == "1"
			shouldDL                  = shouldExtractBpm
			shouldDoSomething         = shouldExtractBpm || shouldDetectRelationships || shouldResyncSoundCloud
		)
		if !shouldDoSomething {
			svc.error404Page(box)(w, r)
			return
		}

		// common init
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
		post, err := svc.storage.GetCustomPost(postSlug)
		if err != nil {
			svc.error404Page(box)(w, r)
			return
		}
		if !data.IsAdmin && data.User.ID != post.Author.ID {
			svc.error404Page(box)(w, r)
			return
		}

		// dl file
		var dl *Download
		if shouldDL {
			var err error
			dl, err = DownloadPost(post, false)
			if err != nil {
				svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
				return
			}
			svc.logger.Debug("file downloaded", zap.String("path", dl.Path))
		}

		// resync soundcloud
		if shouldResyncSoundCloud {
			svc.error404Page(box)(w, r)
			return
		}

		// extract bpm
		if shouldExtractBpm {
			bpm, err := ExtractBPM(dl.Path)
			if err != nil {
				svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
				return
			}
			svc.logger.Debug("BPM extracted", zap.Float64("bpm", bpm))
			post.BPM = bpm
			err = svc.storage.UpdatePost(post)
			if err != nil {
				svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
				return
			}
		}

		if shouldDetectRelationships {
			// FIXME: support more relationship kinds

			err := svc.rwdb().Transaction(func(tx *gorm.DB) error {
				// FIXME: avoid delete/recreate associations if they didn't changed

				body := post.SafeTitle() + "\n\n" + post.SafeDescription()

				if err := tx.Model(&post).Association("RelationshipsAsSource").Clear(); err != nil {
					return err
				}
				if err := tx.Model(&post).Association("RelationshipsAsTarget").Clear(); err != nil {
					return err
				}

				for _, match := range featRegex.FindAllStringSubmatch(body, -1) {
					target := strings.ToLower(strings.TrimSpace(match[len(match)-1]))
					user, err := svc.storage.GetUserBySlug(target)
					if err != nil {
						svc.logger.Debug("cannot find the featured artist in DB", zap.Error(err))
						continue
					}

					if err := tx.Model(&post).Association("RelationshipsAsSource").Append(&sgtmpb.Relationship{
						SourcePostID: post.ID,
						// todo: this may lead to nil pointer? Moul?
						TargetUserID: user.ID,
						Kind:         sgtmpb.Relationship_FeaturingUserKind,
					}); err != nil {
						return err
					}
				}
				return nil
			})
			if err != nil {
				svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
				return
			}
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
			post, err := svc.storage.GetCustomPost(postSlug)
			if err != nil {
				svc.error404Page(box)(w, r)
				return
			}
			data.PostEdit.Post = post
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
				fields["title"] = strings.TrimSpace(r.Form.Get("title"))
				fields["body"] = strings.TrimSpace(r.Form.Get("body"))
				fields["lyrics"] = strings.TrimSpace(r.Form.Get("lyrics"))
				if data.PostEdit.Post.Provider == sgtmpb.Provider_IPFS {
					fields["title"] = r.Form.Get("title")
				}
				return fields
			}
			fields := validate()
			if fields != nil {
				err = svc.storage.GenericUpdatePost(fields)
				if err != nil {
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

func (svc *Service) postDownloadPage(box *packr.Box) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		postSlug := chi.URLParam(r, "post_slug")
		post, err := svc.storage.GetCustomPost(postSlug)
		if err != nil {
			svc.error404Page(box)(w, r)
			return
		}

		if post.MIMEType != "" {
			w.Header().Set("Content-Type", post.MIMEType)
		}
		reader, err := StreamPost(&svc.ipfs, post)
		if err != nil {
			svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
			return
		}
		defer reader.Close()
		http.ServeContent(w, r, "", time.Time{}, reader)
	}
}
