package sgtm

import (
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	packr "github.com/gobuffalo/packr/v2"
	"github.com/yanatan16/golang-soundcloud/soundcloud"
	"go.uber.org/zap"
	"moul.io/godev"
	"moul.io/sgtm/pkg/sgtmpb"
)

func (svc *Service) newPage(box *packr.Box) func(w http.ResponseWriter, r *http.Request) {
	tmpl := loadTemplates(box, "base.tmpl.html", "new.tmpl.html")
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
			validate := func() *sgtmpb.Post {
				if err := r.ParseMultipartForm(25 * 1024 * 1024); err != nil {
					data.Error = err.Error()
					return nil
				}

				fileURL := r.Form.Get("url")

				if r.Form.Get("url") == "" {
					// get file from upload if possible
					file, header, err := r.FormFile("upload")
					if err != nil {
						data.Error = err.Error()
						return nil
					}
					defer file.Close()

					fmt.Println("header", header.Header)

					// stream file to ipfs and get cid
					fmt.Println("uploading to ipfs")
					cid, err := svc.ipfs.add(file)
					if err != nil {
						data.Error = err.Error()
						return nil
					}
					fmt.Println("done uploading to ipfs")

					// check if track already exists
					{
						var alreadyExists sgtmpb.Post
						err := svc.rodb().
							Model(&sgtmpb.Post{}).
							Where(sgtmpb.Post{IPFSCID: cid}).
							First(&alreadyExists).
							Error
						if err == nil && alreadyExists.ID != 0 {
							data.New.URLInvalidMsg = fmt.Sprintf(`This track already exists: <a href="/post/%d">%s</a>.`, alreadyExists.ID, alreadyExists.Title)
							return nil
						}
					}

					mimeType := header.Header["Content-Type"][0]
					fmt.Println("MIME", mimeType)
					// FIXME: check that mimeType starts with audio/ maybe

					// TODO: run mimetype detection on file maybe
					filenameParts := strings.Split(header.Filename, ".")
					ext := "blob"
					filename := header.Filename
					numParts := len(filenameParts)
					if numParts >= 2 {
						ext = filenameParts[numParts-1]
						filename = strings.Join(filenameParts[:numParts-1], ".")
					}

					return &sgtmpb.Post{
						Kind:               sgtmpb.Post_TrackKind,
						Visibility:         sgtmpb.Visibility_Public,
						AuthorID:           data.User.ID,
						Slug:               "",
						Title:              filename,
						SortDate:           time.Now().UnixNano(),
						URL:                "https://ipfs.io/ipfs/" + cid,
						Provider:           sgtmpb.Provider_IPFS,
						IPFSCID:            cid,
						MIMEType:           mimeType,
						SizeBytes:          header.Size,
						FileExtension:      ext,
						AttachmentFilename: header.Filename,
					}
				}

				// FIXME: check if valid SoundCloud link
				post := sgtmpb.Post{
					Kind:       sgtmpb.Post_TrackKind,
					Visibility: sgtmpb.Visibility_Public,
					AuthorID:   data.User.ID,
					Slug:       "",
					Title:      "",
					SortDate:   time.Now().UnixNano(),
				}

				u, err := url.Parse(fileURL)
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

					// check if track already exists
					{
						var alreadyExists sgtmpb.Post
						err := svc.rodb().
							Model(&post).
							Where(sgtmpb.Post{SoundCloudID: post.SoundCloudID}).
							First(&alreadyExists).
							Error
						if err == nil && alreadyExists.ID != 0 {
							data.New.URLInvalidMsg = fmt.Sprintf(`This track already exists: <a href="%s">%s</a>.`, alreadyExists.CanonicalURL(), alreadyExists.Title)
							return nil
						}
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

					post.ProviderMetadata = godev.JSON(track)
					post.Title = track.Title
					createdAt, err := time.Parse("2006/01/02 15:04:05 +0000", track.CreatedAt)
					if err == nil {
						post.ProviderCreatedAt = createdAt.UnixNano()
						post.SortDate = createdAt.UnixNano()
					}
					tags := append(
						[]string{track.Genre},
						strings.Split(track.TagList, " ")...,
					)
					post.Tags = strings.Join(tags, ", ")
					post.Duration = track.Duration
					post.ArtworkURL = strings.ReplaceAll(track.ArtworkUrl, "-large.jpg", "-t500x500.jpg")
					post.ISRC = track.ISRC
					post.BPM = track.Bpm
					post.KeySignature = track.KeySignature
					post.ProviderDescription = track.Description
					/*
						post.Body = track.Description
						post.WaveformURL = track.WaveformURL
						post.License = track.License
						track.User
					*/
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
				if err := svc.rwdb().Create(&post).Error; err != nil {
					svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
					return
				}
				svc.logger.Debug("new post", zap.Any("post", post))
				http.Redirect(w, r, post.CanonicalURL(), http.StatusFound)
				return
			}
		}
		// end of custom
		if svc.opts.DevMode {
			tmpl = loadTemplates(box, "base.tmpl.html", "new.tmpl.html")
		}
		data.Duration = time.Since(started)
		if err := tmpl.Execute(w, &data); err != nil {
			svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
			return
		}
	}
}
