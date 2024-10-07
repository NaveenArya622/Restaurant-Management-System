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

		/*
			need to fetch archived_at form user_session table
			and check whether its is null or not and if the session arhcived_at is not null means the user has
			logged out so send message and status code 401 unauthorized
		*/

		// keep only necessary things in context area

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
