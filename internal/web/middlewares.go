package web

import (
	"net/http"

	"gitea.com/go-chi/session"
	"github.com/rs/zerolog/hlog"
	"gitlab.com/kabes/go-gpo/internal/common"
	"gitlab.com/kabes/go-gpo/internal/server/srvsupport"
)

func newAuthenticatedOnlyWeb(webroot string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := hlog.FromRequest(r)
			sess := session.GetSession(r)
			user := srvsupport.SessionUser(sess)

			logger.Debug().Str("session_user", user).Str("sid", sess.ID()).
				Msgf("AuthenticatedOnly: check user_name=%s sid=%s", user, sess.ID())

			if user != "" {
				next.ServeHTTP(w, r.WithContext(common.ContextWithUser(r.Context(), user)))

				return
			}

			http.Redirect(w, r, webroot+"/web/auth/login", http.StatusFound)
		})
	}
}
