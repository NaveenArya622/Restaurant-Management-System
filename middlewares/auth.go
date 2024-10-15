package middlewares

import (
	"context"
	"net/http"
	"rms/database/dbHelper"
	"rms/models"
	"rms/utils"
	"strings"

	"github.com/sirupsen/logrus"
)

type ContextKeys string

const (
	userContext ContextKeys = "__userContext"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.Split(r.Header.Get("authorization"), " ")[1]
		jwtErr := utils.ParseJwtToken(token)
		if jwtErr != nil {
			logrus.WithError(jwtErr).Errorf("Failed to get user with token: %s", token)
			utils.RespondError(w, http.StatusUnauthorized, jwtErr, "Invalid Token")
			return
		}
		user, err := dbHelper.GetUserBySession(token)
		if err != nil || user == nil {
			logrus.WithError(err).Errorf("Failed to get user with token: %s", token)
			utils.RespondError(w, http.StatusUnauthorized, err, "Failed to get user with token.")
			return
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

func ShouldHaveRole(role models.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := UserContext(r)
			if user == nil {
				logrus.Errorf("Failed to get user: %s", user)
				w.WriteHeader(http.StatusForbidden)
				return
			}
			if user.CurrentRole == role {
				next.ServeHTTP(w, r)
				return
			}
			logrus.Errorf("Failed to invalid UserRole: %s, accepted: %s", user.CurrentRole, role)
			w.WriteHeader(http.StatusForbidden)
		})
	}
}
