package srvsupport

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/render"
	"github.com/rs/zerolog"
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
		logger.Error().Err(err).Msg("encode json failed")

		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
