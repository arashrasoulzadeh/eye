package handler

import "net/http"

// AdminAuth wraps next with a check that the request carries the configured
// admin API key in the x-admin-key header, rejecting anything else with 401.
func AdminAuth(adminKey string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("x-admin-key")
		if key == "" || key != adminKey {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
