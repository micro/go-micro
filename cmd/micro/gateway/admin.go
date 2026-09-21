package gateway

import (
	"encoding/json"
	"net/http"

	"go-micro.dev/v6/store"
)

// adminRequired separates administrative control from tool invocation scopes.
// Both the signed role and the current stored role must authorize administration.
func adminRequired(st store.Store) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return authRequired(st)(func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			if tokenMatches(token) {
				next(w, r)
				return
			}
			claims, err := ParseJWT(token)
			if err == nil && claims["type"] == "admin" {
				id, _ := claims["sub"].(string)
				records, readErr := st.Read("auth/" + id)
				if readErr == nil && id != "" && len(records) == 1 {
					var account Account
					if json.Unmarshal(records[0].Value, &account) == nil && account.ID == id && account.Type == "admin" {
						next(w, r)
						return
					}
				}
			}
			http.Error(w, "Forbidden", http.StatusForbidden)
		})
	}
}
