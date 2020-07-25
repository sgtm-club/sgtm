package sgtm

import (
	"net/http"
	"time"

	packr "github.com/gobuffalo/packr/v2"
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
			if err := svc.rwdb().Create(&viewEvent).Error; err != nil {
				data.Error = "Cannot write activity: " + err.Error()
			} else {
				svc.logger.Debug("new view open", zap.Any("event", &viewEvent))
			}
		}

		// events
		{
			type result struct {
				Kind     sgtmpb.Post_Kind
				Quantity int64
			}
			var results []result
			err := svc.rodb().
				Model(&sgtmpb.Post{}).
				//Where(sgtmpb.Post{Visibility: sgtmpb.Visibility_Public}).
				Select(`kind, count(*) as quantity`).
				Group("kind").
				Find(&results).
				Error
			if err != nil {
				data.Error = "Cannot fetch events: " + err.Error()
			} else {
				for _, result := range results {
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
			var result struct {
				TotalDuration int64
			}
			err := svc.rodb().
				Model(&sgtmpb.Post{}).
				Select("sum(duration) as total_duration").
				Where(sgtmpb.Post{
					Kind: sgtmpb.Post_TrackKind,
					//Visibility: sgtmpb.Visibility_Public,
				}).
				First(&result).
				Error
			if err != nil {
				data.Error = "Cannot fetch last track durations: " + err.Error()
			}
			data.Open.Count.TotalDuration = time.Duration(result.TotalDuration) * time.Millisecond
		}

		// uploads by weekday
		{
			type result struct {
				Weekday  int64
				Quantity int64
			}
			var results []result
			err := svc.rodb().Model(&sgtmpb.Post{}).
				Where(&sgtmpb.Post{Kind: sgtmpb.Post_TrackKind}).
				Select(`strftime("%w", sort_date/1000000000, "unixepoch") as weekday , count(*) as quantity`).
				Group("weekday").Find(&results).
				Error
			if err != nil {
				data.Error = "Cannot fetch uploads by weekday: " + err.Error()
			}
			data.Open.UploadsByWeekday = make([]int64, 7)
			for _, result := range results {
				data.Open.UploadsByWeekday[result.Weekday] = result.Quantity
			}
		}

		// last activities
		{
			err := svc.rodb().
				Preload("Author").
				Preload("TargetPost").
				Preload("TargetUser").
				Order("created_at desc").
				Where("NOT (author_id == ? AND kind IN (?))", moulID, []sgtmpb.Post_Kind{
					sgtmpb.Post_ViewHomeKind,
					sgtmpb.Post_ViewOpenKind,
				}). // filter admin recurring actions
				//Where("author_id != 0"). // filter anonymous
				Where("kind NOT IN (?)", []sgtmpb.Post_Kind{
					sgtmpb.Post_LinkDiscordAccountKind,
					//sgtmpb.Post_LoginKind,
				}).
				Limit(42).
				Find(&data.Open.LastActivities).
				Error
			if err != nil {
				data.Error = "Cannot fetch last activities: " + err.Error()
			}
		}

		// track drafts
		{
			err := svc.rodb().
				Model(&sgtmpb.Post{}).
				Where(sgtmpb.Post{
					Kind:       sgtmpb.Post_TrackKind,
					Visibility: sgtmpb.Visibility_Draft,
				}).
				Count(&data.Open.Count.TrackDrafts).
				Error
			if err != nil {
				data.Error = "Cannot fetch last track drafts: " + err.Error()
			}
		}
		// users
		{
			err := svc.rodb().
				Model(&sgtmpb.User{}).
				Count(&data.Open.Count.Users).
				Error
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
