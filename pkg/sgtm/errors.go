package sgtm

import (
	"net/http"

	"github.com/go-chi/render"
	"go.uber.org/zap"
)

func (svc *Service) errRender(w http.ResponseWriter, r *http.Request, err error, status int) {
	renderer := errToResponse(err, status)
	svc.logger.
		WithOptions(zap.AddCallerSkip(1)).
		Warn(
			"user error",
			zap.String("title", renderer.Title),
			zap.Error(err),
		)
	if err := render.Render(w, r, renderer); err != nil {
		svc.logger.Warn("cannot render error", zap.Error(err))
	}
}

func errToResponse(err error, status int) *errResponse {
	// FIXME: if DevMode, print stacktrace
	if status == 0 {
		status = http.StatusUnprocessableEntity
	}
	return &errResponse{
		Type:     "about:blank",
		Title:    http.StatusText(status),
		Status:   status,
		Detail:   err.Error(),
		Instance: "",
	}
}

// based on github.com/moogar0880/problems.DefaultProblem
type errResponse struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status,omitempty"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
}

func (e *errResponse) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, e.Status)
	return nil
}
