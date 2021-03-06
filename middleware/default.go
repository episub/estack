package middleware

import (
	"context"
	"net/http"

	"github.com/episub/estack/store"
	"github.com/episub/estack/validate"
	"github.com/episub/estack/vars"
)

// DefaultMW Sets up items needed for most requests
// - Adds a data object to the context, used for passing data through to OPA requests
// - Sets validation context
func DefaultMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), vars.SharedData, store.NewDataStore())
		ctx = validate.SetContext(r.Context())
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
