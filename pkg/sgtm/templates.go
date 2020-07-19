package sgtm

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	sprig "github.com/Masterminds/sprig/v3"
	humanize "github.com/dustin/go-humanize"
	packr "github.com/gobuffalo/packr/v2"
	striptags "github.com/grokify/html-strip-tags-go"
	"github.com/hako/durafmt"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"moul.io/sgtm/pkg/sgtmpb"
)

func (svc *Service) newTemplateData(w http.ResponseWriter, r *http.Request) (*templateData, error) {
	data := templateData{
		Title:   "SGTM",
		Date:    time.Now(),
		Opts:    svc.opts.Filtered(),
		Lang:    "en", // FIXME: dynamic
		Request: r,
		Service: svc,
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
		if err := svc.db.
			Preload("RecentPosts", func(db *gorm.DB) *gorm.DB {
				return db.
					Where("kind IN (?)", []sgtmpb.Post_Kind{sgtmpb.Post_TrackKind}).
					Order("created_at desc").
					Limit(3)
			}).
			First(&user, data.Claims.Session.UserID).
			Error; err != nil {
			svc.logger.Warn("load user from DB", zap.Error(err))
		}
		data.User = &user
		data.UserID = user.ID
		data.IsAdmin = user.ID == 1280639244955553792
		//w.Header().Set("SGTM-User-ID", fmt.Sprintf("%d", user.ID))
		w.Header().Set("SGTM-User-Slug", user.Slug)
	} else {
		w.Header().Set("SGTM-User-Slug", "-")
	}

	return &data, nil
}

func loadTemplates(box *packr.Box, filenames ...string) *template.Template {
	allInOne := ""
	templateName := ""
	for _, filename := range filenames {
		src, err := box.FindString("page_" + filename)
		if err != nil {
			panic(err)
		}
		allInOne += strings.TrimSpace(src) + "\n"
		templateName += filename
	}
	allInOne = strings.TrimSpace(allInOne)
	funcmap := sprig.FuncMap()
	funcmap["fromUnixNano"] = func(input int64) time.Time {
		return time.Unix(0, input)
	}
	funcmap["prettyURL"] = func(input string) string {
		u, err := url.Parse(input)
		if err != nil {
			return ""
		}
		if len(u.Path) > 10 {
			u.Path = u.Path[0:7] + "..."
		}
		shorten := fmt.Sprintf("%s%s", u.Host, u.Path)
		shorten = strings.TrimRight(shorten, "/")
		shorten = strings.TrimLeft(shorten, "www.")
		return shorten
	}
	funcmap["newline"] = func() string {
		return "\n"
	}
	funcmap["prettyAgo"] = func(input time.Time) string {
		return humanize.RelTime(input, time.Now(), "ago", "in the future(!?)")
	}
	funcmap["prettyDuration"] = func(input time.Duration) string {
		input = input.Round(time.Second)
		str := durafmt.Parse(input).LimitFirstN(2).String()
		str = strings.Replace(str, " ", "", -1)
		str = strings.Replace(str, "minutes", "m", -1)
		str = strings.Replace(str, "minute", "m", -1)
		str = strings.Replace(str, "hours", "h", -1)
		str = strings.Replace(str, "hour", "h", -1)
		str = strings.Replace(str, "seconds", "s", -1)
		str = strings.Replace(str, "second", "s", -1)

		return str
	}
	funcmap["prettyDate"] = func(input time.Time) string {
		return input.Format("2006-01-02 15:04")
	}
	funcmap["noescape"] = func(str string) template.HTML {
		return template.HTML(str)
	}
	funcmap["stripTags"] = striptags.StripTags
	funcmap["urlencode"] = url.PathEscape
	funcmap["plus1"] = func(x int) int {
		return x + 1
	}
	tmpl, err := template.New(templateName).Funcs(funcmap).Parse(allInOne)
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
	IsAdmin  bool
	User     *sgtmpb.User
	UserID   int64
	Error    string
	Service  *Service      `json:"-"`
	Request  *http.Request `json:"-"`

	// specific

	RSS struct {
		LastTracks []*sgtmpb.Post
	}
	Home struct {
		LastTracks []*sgtmpb.Post
		LastUsers  []*sgtmpb.User
	} `json:"Home,omitempty"`
	Settings struct {
	} `json:"Settings,omitempty"`
	Profile struct {
		User       *sgtmpb.User
		LastTracks []*sgtmpb.Post
		Stats      struct {
			Tracks int64
			// Drafts int64
		}
		CalendarHeatmap map[int64]int64
	} `json:"Profile,omitempty"`
	Open struct {
		Users            int64
		Tracks           int64
		TrackDrafts      int64
		TotalDuration    time.Duration
		UploadsByWeekday []int64
		LastActivities   []*sgtmpb.Post
	} `json:"Open,omitempty"`
	New struct {
		URLValue      string
		URLInvalidMsg string
	} `json:"New,omitempty"`
	Post struct {
		Post *sgtmpb.Post
	} `json:"Post,omitempty"`
	PostEdit struct {
		Post *sgtmpb.Post
	} `json:"PostEdit,omitempty"`
}
