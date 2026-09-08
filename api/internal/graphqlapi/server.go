package graphqlapi

import (
	"crypto/subtle"
	"net/http"

	gqlhandler "github.com/graphql-go/handler"

	"endonend/api/internal/crawler"
	"endonend/api/internal/store"
)

// NewPublicHandler serves the versioned, read-only public API at
// /v1/graphql: browse view and playback page data, safe for any listener.
func NewPublicHandler(s *store.Store) http.Handler {
	h := gqlhandler.New(&gqlhandler.Config{Schema: &PublicSchema, Pretty: false, GraphiQL: false})
	return withCORS(injectContext(h, s, nil))
}

// NewAdminHandler serves the versioned admin API at /v1/admin/graphql:
// everything the public API has, plus the onboarding mutations, gated by a
// shared secret and never linked from any public page, per
// KB/0010-mvp-scope.md's admin console design.
func NewAdminHandler(s *store.Store, c *crawler.Crawler, secret string) http.Handler {
	h := gqlhandler.New(&gqlhandler.Config{Schema: &AdminSchema, Pretty: false, GraphiQL: false})
	return withCORS(requireAdminSecret(secret, injectContext(h, s, c)))
}

// withCORS lets the web app (served from its own origin, a different
// port than this API) call it from the browser at all. It has to be the
// outermost wrapper, ahead of requireAdminSecret: a browser's CORS
// preflight OPTIONS request never carries X-Admin-Token, so the auth check
// would otherwise reject every preflight before the real POST is ever
// sent. Origin is wildcarded rather than allowlisted since this API's real
// access boundary is the admin token itself (a header, not a cookie), not
// the browser's same-origin policy.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Admin-Token")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func injectContext(next http.Handler, s *store.Store, c *crawler.Crawler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := withStore(r.Context(), s)
		if c != nil {
			ctx = withCrawler(ctx, c)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// requireAdminSecret checks the X-Admin-Token header against secret using a
// constant-time comparison, so response timing can't leak how much of the
// token a caller guessed correctly. This is the "founder-only, not linked
// from any public page" gate KB/0010-mvp-scope.md describes for the admin
// console; it is not a multi-user auth system.
func requireAdminSecret(secret string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("X-Admin-Token")
		if subtle.ConstantTimeCompare([]byte(got), []byte(secret)) != 1 || secret == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
