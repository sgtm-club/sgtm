package sgtm

import (
	"fmt"
	"html"
	"html/template"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	sprig "github.com/Masterminds/sprig/v3"
	"github.com/go-chi/chi"
	packr "github.com/gobuffalo/packr/v2"
	"github.com/yanatan16/golang-soundcloud/soundcloud"
	"go.uber.org/zap"
	"moul.io/godev"
	"moul.io/sgtm/pkg/sgtmpb"
)

func (svc *Service) indexPage(box *packr.Box) func(w http.ResponseWriter, r *http.Request) {
	tmpl := loadTemplate(box, "_layouts/index.tmpl.html")
	return func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		data, err := svc.newTemplateData(r)
		if err != nil {
			svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
			return
		}
		// custom
		// end of custom
		if svc.opts.DevMode {
			tmpl = loadTemplate(box, "_layouts/index.tmpl.html")
		}
		data.Duration = time.Since(started)
		if err := tmpl.ExecuteTemplate(w, "base", &data); err != nil {
			svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
			return
		}
	}
}

func (svc *Service) settingsPage(box *packr.Box) func(w http.ResponseWriter, r *http.Request) {
	tmpl := loadTemplate(box, "_layouts/settings.tmpl.html")
	return func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		data, err := svc.newTemplateData(r)
		if err != nil {
			svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
			return
		}
		// custom
		if data.User == nil {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}
		// end of custom
		if svc.opts.DevMode {
			tmpl = loadTemplate(box, "_layouts/settings.tmpl.html")
		}
		data.Duration = time.Since(started)
		if err := tmpl.Execute(w, &data); err != nil {
			svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
			return
		}
	}
}

func (svc *Service) newPage(box *packr.Box) func(w http.ResponseWriter, r *http.Request) {
	tmpl := loadTemplate(box, "_layouts/new.tmpl.html")
	return func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		data, err := svc.newTemplateData(r)
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
			validate := func() *sgtmpb.Post {
				if err := r.ParseForm(); err != nil {
					data.Error = err.Error()
					return nil
				}
				if r.Form.Get("url") == "" {
					data.New.URLInvalidMsg = "Please specify a track link."
					return nil
				}
				data.New.URLValue = r.Form.Get("url")

				// FIXME: check if valid SoundCloud link
				post := sgtmpb.Post{
					Kind:       sgtmpb.Post_TrackKind,
					Visibility: sgtmpb.Visibility_Public,
					AuthorID:   data.User.ID,
					Slug:       "",
					Title:      "",
				}

				u, err := url.Parse(r.Form.Get("url"))
				if err != nil {
					data.Error = fmt.Sprintf("Parse URL: %s", err.Error())
					return nil
				}
				switch u.Host {
				case "soundcloud.com":
					sc := soundcloud.Api{ClientId: svc.opts.SoundCloudClientID}
					u, err := sc.Resolve(u.String())
					if err != nil {
						data.New.URLInvalidMsg = "This URL does not exist on SoundCloud.com."
						return nil
					}
					re := regexp.MustCompile(`/tracks/(.*).json`)
					matches := re.FindStringSubmatch(u.Path)
					if len(matches) != 2 {
						data.New.URLInvalidMsg = "Invalid SoundCloud track link."
						return nil
					}
					post.SoundCloudKind = sgtmpb.Post_SoundCloudTrack
					post.SoundCloudID, err = strconv.ParseUint(matches[1], 10, 64)
					if err != nil {
						data.New.URLInvalidMsg = fmt.Sprintf("Parse track ID: %s.", err.Error())
						return nil
					}

					post.SoundCloudSecretToken = u.Query().Get("secret_token")
					params := url.Values{}
					if post.SoundCloudSecretToken != "" {
						params.Set("secret_token", post.SoundCloudSecretToken)
					}
					track, err := sc.Track(post.SoundCloudID).Get(params)
					if err != nil {
						data.New.URLInvalidMsg = fmt.Sprintf("Fetch track info from SoundCloud: %s.", err.Error())
						return nil
					}

					fmt.Println(godev.PrettyJSON(track))

					post.ProviderMetadata = godev.JSON(track)
					post.Title = track.Title
					createdAt, err := time.Parse("2006/01/02 15:04:05 +0000", track.CreatedAt)
					if err == nil {
						post.ProviderCreatedAt = createdAt.UnixNano()
					}
					post.Genre = track.Genre
					post.Duration = track.Duration
					post.ArtworkURL = track.ArtworkUrl
					post.ISRC = track.ISRC
					post.BPM = track.Bpm
					post.KeySignature = track.KeySignature
					post.ProviderDescription = track.Description
					post.Body = track.Description
					/*post.Tags = track.TagList
					post.WaveformURL = track.WaveformURL
					post.License = track.License
					track.User*/
					if track.Downloadable {
						post.DownloadURL = track.DownloadUrl
					}
					post.URL = track.PermalinkUrl
					post.Provider = sgtmpb.Provider_SoundCloud
				default:
					data.New.URLInvalidMsg = fmt.Sprintf("Unsupported provider: %s.", html.EscapeString(u.Host))
					return nil
				}

				if r.Form.Get("submit") == "draft" {
					post.Visibility = sgtmpb.Visibility_Draft
				}
				return &post
			}
			post := validate()
			if post != nil {
				if err := svc.db.Create(&post).Error; err != nil {
					svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
					return
				}
				svc.logger.Debug("new post", zap.Any("post", post))
				http.Redirect(w, r, post.CanonicalURL(), http.StatusTemporaryRedirect)
				return
			}
		}
		// end of custom
		if svc.opts.DevMode {
			tmpl = loadTemplate(box, "_layouts/new.tmpl.html")
		}
		data.Duration = time.Since(started)
		if err := tmpl.Execute(w, &data); err != nil {
			svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
			return
		}
	}
}

func (svc *Service) error404Page(box *packr.Box) func(w http.ResponseWriter, r *http.Request) {
	tmpl := loadTemplate(box, "_layouts/error404.tmpl.html")
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
			tmpl = loadTemplate(box, "_layouts/error404.tmpl.html")
		}
		data.Duration = time.Since(started)
		if err := tmpl.Execute(w, &data); err != nil {
			svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
			return
		}
	}
}

func (svc *Service) profilePage(box *packr.Box) func(w http.ResponseWriter, r *http.Request) {
	tmpl := loadTemplate(box, "_layouts/profile.tmpl.html")
	return func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		data, err := svc.newTemplateData(r)
		if err != nil {
			svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
			return
		}
		// custom

		// load profile
		{
			userSlug := chi.URLParam(r, "user_slug")
			var user sgtmpb.User
			if err := svc.db.
				Where(sgtmpb.User{Slug: userSlug}).
				First(&user).
				Error; err != nil {
				svc.error404Page(box)(w, r)
				return
			}
			data.Profile.User = &user
		}

		// tracks
		{
			query := svc.db.
				Model(&sgtmpb.Post{}).
				Where(sgtmpb.Post{
					AuthorID:   data.Profile.User.ID,
					Kind:       sgtmpb.Post_TrackKind,
					Visibility: sgtmpb.Visibility_Public,
				})
			if err := query.Count(&data.Profile.Stats.Tracks).Error; err != nil {
				data.Error = "Cannot fetch last tracks: " + err.Error()
			}
			if data.Profile.Stats.Tracks > 0 {
				if err := query.
					Order("created_at desc").
					Limit(10). // FIXME: pagination
					Find(&data.Profile.LastTracks).
					Error; err != nil {
					data.Error = "Cannot fetch last tracks: " + err.Error()
				}
			}
			for _, track := range data.Profile.LastTracks {
				track.ApplyDefaults()
			}
		}
		// end of custom
		if svc.opts.DevMode {
			tmpl = loadTemplate(box, "_layouts/profile.tmpl.html")
		}
		data.Duration = time.Since(started)
		if err := tmpl.Execute(w, &data); err != nil {
			svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
			return
		}
	}
}

func (svc *Service) postPage(box *packr.Box) func(w http.ResponseWriter, r *http.Request) {
	tmpl := loadTemplate(box, "_layouts/post.tmpl.html")
	return func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		data, err := svc.newTemplateData(r)
		if err != nil {
			svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
			return
		}
		// custom
		postSlug := chi.URLParam(r, "post_slug")
		query := svc.db.Preload("Author")
		id, err := strconv.ParseInt(postSlug, 10, 64)
		if err == nil {
			query = query.Where(sgtmpb.Post{ID: id})
		} else {
			query = query.Where(sgtmpb.Post{Slug: postSlug})
		}
		var post sgtmpb.Post
		if err := query.First(&post).Error; err != nil {
			svc.error404Page(box)(w, r)
			return
		}
		data.Post.Post = &post
		data.Post.Post.ApplyDefaults()

		// end of custom
		if svc.opts.DevMode {
			tmpl = loadTemplate(box, "_layouts/post.tmpl.html")
		}
		data.Duration = time.Since(started)
		if err := tmpl.Execute(w, &data); err != nil {
			svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
			return
		}
	}
}

func (svc *Service) newTemplateData(r *http.Request) (*templateData, error) {
	data := templateData{
		Title:   "SGTM",
		Date:    time.Now(),
		Opts:    svc.opts.Filtered(),
		Lang:    "en", // FIXME: dynamic
		Request: r,
	}
	if svc.opts.DevMode {
		data.Title += " (dev)"
	}

	if cookie, err := r.Cookie(oauthTokenCookie); err == nil {
		data.JWTToken = cookie.Value
		var err error
		data.Claims, err = svc.parseJWTToken(data.JWTToken)
		if err != nil {
			return nil, fmt.Errorf("parse jwt token: %w", err)
		}
		var user sgtmpb.User
		if err := svc.db.First(&user, data.Claims.Session.UserID).Error; err != nil {
			svc.logger.Warn("load user from DB", zap.Error(err))
		}
		data.User = &user
	}

	return &data, nil
}

func loadTemplate(box *packr.Box, filepath string) *template.Template {
	src, err := box.FindString(filepath)
	if err != nil {
		panic(err)
	}
	base, err := box.FindString("_layouts/base.tmpl.html")
	if err != nil {
		panic(err)
	}
	allInOne := strings.Join([]string{
		strings.TrimSpace(src),
		strings.TrimSpace(base),
	}, "\n")
	funcmap := sprig.FuncMap()
	funcmap["fromUnixNano"] = func(input int64) time.Time {
		return time.Unix(0, input)
	}
	tmpl, err := template.New("index").Funcs(funcmap).Parse(allInOne)
	if err != nil {
		panic(err)
	}
	return tmpl
}

type templateData struct {
	// common

	Title    string
	Date     time.Time
	JWTToken string
	Claims   *jwtClaims
	Duration time.Duration
	Opts     Opts
	Lang     string
	User     *sgtmpb.User
	Error    string
	Request  *http.Request `json:"-"`

	// specific

	Index    struct{} `json:"Index,omitempty"`
	Settings struct{} `json:"Settings,omitempty"`
	Profile  struct {
		User       *sgtmpb.User
		LastTracks []*sgtmpb.Post
		Stats      struct {
			Tracks int64
			// Drafts int64
		}
	} `json:"Profile,omitempty"`
	New struct {
		URLValue      string
		URLInvalidMsg string
	} `json:"New,omitempty"`
	Post struct{ Post *sgtmpb.Post } `json:"Post,omitempty"`
}
