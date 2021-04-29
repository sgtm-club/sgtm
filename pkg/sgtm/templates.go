package sgtm

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/dustin/go-humanize"
	"github.com/gobuffalo/packr/v2"
	striptags "github.com/grokify/html-strip-tags-go"
	"github.com/hako/durafmt"
	"github.com/kyokomi/emoji/v2"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday/v2"
	"go.uber.org/zap"

	"moul.io/sgtm/internal/sgtmversion"
	"moul.io/sgtm/pkg/sgtmpb"
)

func (svc *Service) newTemplateData(w http.ResponseWriter, r *http.Request) (*templateData, error) {
	data := templateData{
		Title:            "SGTM",
		Date:             time.Now(),
		Opts:             svc.opts.Filtered(),
		Lang:             "en", // FIXME: dynamic
		Request:          r,
		Service:          svc,
		PageKind:         "other",
		ReleaseVersion:   sgtmversion.Version,
		ReleaseVcsRef:    sgtmversion.VcsRef,
		ReleaseBuildDate: sgtmversion.BuildDate,
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
		var user *sgtmpb.User
		user, err = svc.store.GetUserRecentPost(data.Claims.Session.UserID)
		if err != nil {
			svc.logger.Warn("load user from DB", zap.Error(err))
		}
		if user != nil {
			data.User = user
			data.UserID = user.ID
			data.IsAdmin = user.Role == "admin"
			// w.Header().Set("SGTM-User-ID", fmt.Sprintf("%d", user.ID))
			w.Header().Set("SGTM-User-Slug", user.Slug)
		}
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
	funcmap["markdownify"] = func(input string) template.HTML {
		extensions := blackfriday.CommonExtensions | blackfriday.NoEmptyLineBeforeBlock
		unsafe := blackfriday.Run([]byte(input), blackfriday.WithExtensions(extensions))
		mdHTML := bluemonday.UGCPolicy().SanitizeBytes(unsafe)
		html := fmt.Sprintf(`<div class="markdownify">%s</div>`, string(mdHTML))
		return template.HTML(html)
	}
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
		shorten = strings.TrimPrefix(shorten, "www.")
		return shorten
	}
	funcmap["emojify"] = func(input string) string {
		return emoji.Sprint(input)
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
		str = strings.ReplaceAll(str, " ", "")
		str = strings.ReplaceAll(str, "minutes", "m")
		str = strings.ReplaceAll(str, "minute", "m")
		str = strings.ReplaceAll(str, "hours", "h")
		str = strings.ReplaceAll(str, "hour", "h")
		str = strings.ReplaceAll(str, "seconds", "s")
		str = strings.ReplaceAll(str, "second", "s")

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

	PageKind         string
	Title            string
	Date             time.Time
	JWTToken         string
	Claims           *jwtClaims
	Duration         time.Duration
	Opts             Opts
	Lang             string
	IsAdmin          bool
	User             *sgtmpb.User
	UserID           int64
	Error            string
	Service          *Service      `json:"-"`
	Request          *http.Request `json:"-"`
	ReleaseVersion   string
	ReleaseVcsRef    string
	ReleaseBuildDate string

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
		Count struct {
			Users         int64
			Tracks        int64
			TrackDrafts   int64
			Comments      int64
			PostViews     int64
			Logins        int64
			HomeViews     int64
			OpenViews     int64
			ProfileViews  int64
			TotalDuration time.Duration
		}
		UploadsByWeekday []int64
		LastActivities   []*sgtmpb.Post
	} `json:"Open,omitempty"`
	New struct {
		URLValue      string
		URLInvalidMsg string
	} `json:"New,omitempty"`
	Post struct {
		Post     *sgtmpb.Post
		Comments []*sgtmpb.Post
	} `json:"Post,omitempty"`
	PostEdit struct {
		Post *sgtmpb.Post
	} `json:"PostEdit,omitempty"`
}
