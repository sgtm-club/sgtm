package sgtm

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/moogar0880/problems"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"moul.io/godev"
)

var (
	invalidStateProblem = problems.NewDetailedProblem(http.StatusBadRequest, "invalid state")
	codeExchangeProblem = problems.NewDetailedProblem(http.StatusInternalServerError, "oauth code exchange")
	// internalProblem     = problems.NewDetailedProblem(http.StatusInternalServerError, "internal problem")
)

const (
	oauthTokenCookie = "oauth-token"
)

func (svc *Service) httpAuthLogin(w http.ResponseWriter, r *http.Request) {
	conf := svc.authConfigFromReq(r)
	state := svc.authGenerateState(r)
	http.Redirect(w, r, conf.AuthCodeURL(state), http.StatusTemporaryRedirect)
}

func (svc *Service) httpAuthLogout(w http.ResponseWriter, r *http.Request) {
	cookie := http.Cookie{
		Name: oauthTokenCookie,
	}
	http.SetCookie(w, &cookie)
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func (svc *Service) httpAuthCallback(w http.ResponseWriter, r *http.Request) {
	conf := svc.authConfigFromReq(r)

	// verifiy oauth2 state
	{
		got := r.URL.Query().Get("state")
		expected := svc.authGenerateState(r)
		if expected != got {
			svc.logger.Warn("invalid oauth2 state", zap.String("expected", expected), zap.String("got", got))
			problems.StatusProblemHandler(invalidStateProblem)(w, r)
			return
		}
	}

	// exchange the code
	var token *oauth2.Token
	{
		code := r.URL.Query().Get("code")
		var err error
		token, err = conf.Exchange(context.Background(), code)
		if err != nil {
			svc.logger.Warn("code exchange failed", zap.Error(err))
			problems.StatusProblemHandler(codeExchangeProblem)(w, r)
			return
		}
		cookie := http.Cookie{
			Name:     oauthTokenCookie,
			Value:    token.AccessToken,
			Expires:  token.Expiry,
			HttpOnly: true,
			Path:     "/",
			//Domain:   r.Host,
		}
		fmt.Println(godev.PrettyJSON(cookie))
		http.SetCookie(w, &cookie)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	}

	// FIXME: configure jwt, embed username, email, create account if not exists, set roles in jwt
	/*
		// get user's info
		{
			res, err := conf.Client(context.Background(), token).Get("https://discordapp.com/api/v6/users/@me")
			if err != nil {
				svc.logger.Warn("init discord client", zap.Error(err))
				problems.StatusProblemHandler(internalProblem)(w, r)
				return
			}
			defer res.Body.Close()
			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				svc.logger.Warn("init discord client", zap.Error(err))
				problems.StatusProblemHandler(internalProblem)(w, r)
				return
			}
			_, _ = w.Write(body)
		}
	*/
}

func (svc *Service) authConfigFromReq(r *http.Request) *oauth2.Config {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	hostname := fmt.Sprintf("%s://%s", scheme, r.Host)
	return &oauth2.Config{
		Endpoint: oauth2.Endpoint{
			AuthURL:   "https://discordapp.com/api/oauth2/authorize",
			TokenURL:  "https://discordapp.com/api/oauth2/token",
			AuthStyle: oauth2.AuthStyleInParams,
		},
		Scopes:       []string{"identify", "email"},
		RedirectURL:  hostname + "/auth/callback",
		ClientID:     svc.opts.DiscordClientID,
		ClientSecret: svc.opts.DiscordClientSecret,
	}
}

func (svc *Service) authGenerateState(r *http.Request) string {
	// FIXME: add IP too?
	csum := sha256.Sum256([]byte(r.UserAgent() + svc.opts.DiscordClientSecret))
	return base64.StdEncoding.EncodeToString(csum[:])
}
