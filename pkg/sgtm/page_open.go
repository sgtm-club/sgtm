package sgtm

import (
	"net/http"
	"time"

	"github.com/gobuffalo/packr/v2"
	"go.uber.org/zap"

	"moul.io/sgtm/pkg/sgtmpb"
)

func (svc *Service) openPage(box *packr.Box) func(w http.ResponseWriter, r *http.Request) {
	tmpl := loadTemplates(box, "base.tmpl.html", "open.tmpl.html")
	return func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		data, err := svc.newTemplateData(w, r)
		if err != nil {
			svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
			return
		}
		// custom
		data.PageKind = "open"

		// tracking
		{
			viewEvent := sgtmpb.Post{AuthorID: data.UserID, Kind: sgtmpb.Post_ViewOpenKind}
			if err := svc.storage.PatchPost(&viewEvent); err != nil {
				data.Error = "Cannot write activity: " + err.Error()
			} else {
				svc.logger.Debug("new view open", zap.Any("event", &viewEvent))
			}
		}

		// events
		{
			postByKind, err := svc.storage.GetNumberOfPostsByKind()
			if err != nil {
				data.Error = "Cannot fetch events: " + err.Error()
			} else {
				for _, result := range postByKind {
					switch result.Kind {
					case sgtmpb.Post_TrackKind:
						data.Open.Count.Tracks = result.Quantity
					case sgtmpb.Post_CommentKind:
						data.Open.Count.Comments = result.Quantity
					case sgtmpb.Post_ViewHomeKind:
						data.Open.Count.HomeViews = result.Quantity
					case sgtmpb.Post_ViewPostKind:
						data.Open.Count.PostViews = result.Quantity
					case sgtmpb.Post_ViewProfileKind:
						data.Open.Count.ProfileViews = result.Quantity
					case sgtmpb.Post_ViewOpenKind:
						data.Open.Count.OpenViews = result.Quantity
					}
				}
			}
		}

		// tracks' duration
		{
			totalDuration, err := svc.storage.GetTotalDuration()
			if err != nil {
				data.Error = "Cannot fetch last track durations: " + err.Error()
			}
			data.Open.Count.TotalDuration = time.Duration(totalDuration) * time.Millisecond
		}

		{
			upbyweek, err := svc.storage.GetUploadsByWeek()
			if err != nil {
				data.Error = "Cannot fetch uploads by weekday: " + err.Error()
			}
			data.Open.UploadsByWeekday = make([]int64, 7)
			for _, result := range upbyweek {
				data.Open.UploadsByWeekday[result.Weekday] = result.Quantity
			}
		}

		// last activities
		{
			data.Open.LastActivities, err = svc.storage.GetLastActivities(moulID)
			if err != nil {
				data.Error = "Cannot fetch last activities: " + err.Error()
			}
		}

		// track drafts
		{
			data.Open.Count.TrackDrafts, err = svc.storage.GetNumberOfDraftPosts()
			if err != nil {
				data.Error = "Cannot fetch last track drafts: " + err.Error()
			}
		}
		// users
		{
			data.Open.Count.Users, err = svc.storage.GetNumberOfUsers()
			if err != nil {
				data.Error = "Cannot fetch last users: " + err.Error()
			}
		}
		// end of custom
		if svc.opts.DevMode {
			tmpl = loadTemplates(box, "base.tmpl.html", "open.tmpl.html")
		}
		data.Duration = time.Since(started)
		if err := tmpl.Execute(w, &data); err != nil {
			svc.errRenderHTML(w, r, err, http.StatusUnprocessableEntity)
			return
		}
	}
}
