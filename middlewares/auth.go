package middlewares

import (
	"context"
	"net/http"
	"rms/configuration"
	"rms/database/dbHelper"
	"rms/models"
	"rms/utils"
	"strings"
)

type ContextKeys string

const (
	userContext ContextKeys = "__userContext"
)

func AuthMiddleware(next http.Handler) http.Handler {

	config, err := configuration.GetConfig()
	if err != nil {
		log.Logger.Printf("Error loading .env file")
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.Split(r.Header.Get("authorization"), " ")[1]
		jwtErr := utils.ParseJwtToken(token, config.SessionKey)
		if jwtErr != nil {
			log.Logger.WithError(jwtErr).Errorf("Failed to get user with token: %s", token)
			utils.RespondError(w, http.StatusUnauthorized, jwtErr, "Invalid Token", r.Body)
			return
		}
		user, err := dbHelper.GetUserBySession(token)
		if err != nil || user == nil {
			log.Logger.WithError(err).Errorf("Failed to get user with token: %s", token)
			utils.RespondError(w, http.StatusUnauthorized, err, "Failed to get user with token.", r.Body)
			return
		}
		for _, role := range []models.Role{models.RoleAdmin, models.RoleSubAdmin, models.RoleUser} {
			if role.Contains(user.Roles) {
				user.CurrentRole = role
			}
		}
		ctx := context.WithValue(r.Context(), userContext, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserContext(r *http.Request) *models.User {
	if user, ok := r.Context().Value(userContext).(*models.User); ok && user != nil {
		return user
	}
	return nil
}

func ShouldHaveRole(roles []models.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := UserContext(r)
			if user == nil {
				log.Logger.Errorf("Failed to get user: %s", user)
				w.WriteHeader(http.StatusForbidden)
				return
			}
			for _, role := range roles {
				if role.Contains(user.Roles) {
					user.CurrentRole = role
					ctx := context.WithValue(r.Context(), userContext, user)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}
			log.Logger.Errorf("Failed to invalid UserRole: %v, accepted: %s", user.Roles, roles)
			w.WriteHeader(http.StatusForbidden)
		})
	}
}
