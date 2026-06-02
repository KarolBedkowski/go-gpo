package srvsupport

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/render"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
)

// RenderJSON marshals 'v' to JSON, automatically escaping HTML and setting the
// Content-Type as application/json.
// based on go-chi/render but not use temporary buffer.
func RenderJSON(w http.ResponseWriter, r *http.Request, v any) {
	w.Header().Set("Content-Type", "application/json")

	ctx := r.Context()

	if status, ok := ctx.Value(render.StatusCtxKey).(int); ok {
		w.WriteHeader(status)
	}

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)

	if err := enc.Encode(v); err != nil {
		logger := zerolog.Ctx(ctx)
		logger.Error().Err(err).Msgf("encode json failed: %s", err)

		var reqid string
		if rid, ok := hlog.IDFromRequest(r); ok {
			reqid = rid.String()
		}

		now := time.Now().Format(time.RFC3339Nano)

		res := struct {
			Error string `json:"error"`
			ReqID string `json:"request_id,omitempty"`
			TS    string `json:"ts,omitempty"`
		}{http.StatusText(http.StatusInternalServerError), reqid, now}

		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, &res)
	}
}
