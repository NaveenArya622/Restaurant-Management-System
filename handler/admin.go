package handler

import (
	"net/http"
	"os"
	"rms/database"
	"rms/database/dbHelper"
	"rms/middlewares"
	"rms/models"
	"rms/utils"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

func RegisterSubAdmin(w http.ResponseWriter, r *http.Request) {
	var body models.RegisterUserBody
	adminCtx := middlewares.UserContext(r)
	if parseErr := utils.ParseBody(r.Body, &body); parseErr != nil {
		logrus.Printf("Failed to parse request body: %s", parseErr)
		utils.RespondError(w, http.StatusBadRequest, parseErr, "Failed to parse request body")
		return
	}
	if len(body.Password) < 6 {
		logrus.Printf("password must be 6 chars long.")
		utils.RespondError(w, http.StatusBadRequest, nil, "password must be 6 chars long")
		return
	}

	if !utils.IsEmailValid(body.Email) {
		logrus.Printf("Invalid Email.")
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Email.")
		return
	}

	exists, existsErr := dbHelper.IsUserRoleExists(body.Email, models.RoleSubAdmin)
	if existsErr != nil {
		logrus.Printf("Failed to check user role existence: %s", existsErr)
		utils.RespondError(w, http.StatusInternalServerError, existsErr, "Failed to check Sub-Admin existence")
		return
	}
	if exists {
		logrus.Printf("Sub-Admin already exists")
		utils.RespondError(w, http.StatusBadRequest, nil, "Sub-Admin already exists")
		return
	}
	hashedPassword, hasErr := utils.HashPassword(body.Password)
	if hasErr != nil {
		logrus.Printf("Failed to secure password: %s", hasErr)
		utils.RespondError(w, http.StatusInternalServerError, hasErr, "Failed to secure password")
		return
	}
	txErr := database.Tx(func(tx *sqlx.Tx) error {
		userID, existsErr := dbHelper.IsUserExists(body.Email)
		if existsErr != nil {
			logrus.Printf("Failed to check user existence: %s", existsErr)
			utils.RespondError(w, http.StatusInternalServerError, existsErr, "Failed to check user existence")
			return existsErr
		}
		if len(userID) > 0 {
			roleErr := dbHelper.CreateUserRole(tx, userID, adminCtx.ID, models.RoleSubAdmin)
			if roleErr != nil {
				logrus.Printf("Failed to create User Role: %s", roleErr)
				return roleErr
			}
		} else {
			userID, saveErr := dbHelper.CreateUser(tx, body.Name, body.Email, hashedPassword)
			if saveErr != nil {
				logrus.Printf("Failed to Save Sub-Admin: %s", saveErr)
				return saveErr
			}
			roleErr := dbHelper.CreateUserRole(tx, userID, adminCtx.ID, models.RoleSubAdmin)
			if roleErr != nil {
				logrus.Printf("Failed to create User Role: %s", roleErr)
				return roleErr
			}
		}
		return nil
	})
	if txErr != nil {
		logrus.Printf("Failed to create SubAdmin: %s", txErr)
		utils.RespondError(w, http.StatusInternalServerError, txErr, "Failed to create user")
		return
	}
	logrus.Printf("SubAdmin Created successfully.")
	utils.RespondJSON(w, http.StatusCreated, models.Message{
		Message: "SubAdmin Created successfully",
	})
}

func GetSubAdmins(w http.ResponseWriter, r *http.Request) {
	Filters := utils.GetFilters(r)
	if Filters.Email != "" && !utils.IsEmailValid(Filters.Email) {
		logrus.Printf("Invalid Filter Email.")
		utils.RespondError(w, http.StatusInternalServerError, nil, "Invalid Filter Email.")
		return
	}
	UserCount, countErr := dbHelper.GetUserCount(models.RoleSubAdmin, Filters)
	if countErr != nil {
		logrus.Printf("Unable to get Users: %s", countErr)
		utils.RespondError(w, http.StatusInternalServerError, countErr, "Unable to get Users")
		return
	}
	subAdmins, err := dbHelper.GetUsers(models.RoleSubAdmin, Filters)
	if err != nil {
		logrus.Printf("Unable to get Sub-Admin: %s", err)
		utils.RespondError(w, http.StatusInternalServerError, err, "Unable to get Sub-Admin")
		return
	}
	logrus.Printf("Get subAdmin successfully.")
	utils.RespondJSON(w, http.StatusCreated, models.GetSubAdmins{
		Message:    "Get subAdmin successfully.",
		SubAdmins:  subAdmins,
		TotalCount: UserCount,
		PageSize:   Filters.PageSize,
		PageNumber: Filters.PageNumber,
	})
}

//comments :- i can't see any use of RegisterAdmin function ? admin can't register

func RegisterAdmin() {
	exists, existsErr := dbHelper.IsUserRoleExists(os.Getenv("ADMIN_EMAIL"), models.RoleAdmin)
	if existsErr != nil {
		logrus.Printf("User Exist: %s", existsErr)
		return
	}
	if exists {
		logrus.Printf("User Exist.")
		return
	}
	hashedPassword, hasErr := utils.HashPassword(os.Getenv("ADMIN_FIRST_PASSWORD"))
	if hasErr != nil {
		logrus.Printf("Unable to Hash Password: %s", existsErr)
		return
	}
	txErr := database.Tx(func(tx *sqlx.Tx) error {
		userID, userExistsErr := dbHelper.IsUserExists(os.Getenv("ADMIN_EMAIL"))
		if userExistsErr != nil {
			logrus.Printf("User Exist: %s", userExistsErr)
			return userExistsErr
		}
		if len(userID) > 0 {
			roleErr := dbHelper.CreateUserRole(tx, userID, userID, models.RoleAdmin)
			if roleErr != nil {
				logrus.Printf("User Role: %s", roleErr)
				return roleErr
			}
		} else {
			userID, saveErr := dbHelper.CreateUser(tx, os.Getenv("ADMIN_NAME"), os.Getenv("ADMIN_EMAIL"), hashedPassword)
			if saveErr != nil {
				logrus.Printf("User Save: %s", saveErr)
				return saveErr
			}
			roleErr := dbHelper.CreateUserRole(tx, userID, userID, models.RoleAdmin)
			if roleErr != nil {
				logrus.Printf("User My Role: %s", roleErr)
				return roleErr
			}
		}
		return nil
	})
	if txErr != nil {
		return
	}
}

func RemoveSubAdmin(w http.ResponseWriter, r *http.Request) {
	subAdminId := chi.URLParam(r, "subAdminId")
	adminCtx := middlewares.UserContext(r)
	err := dbHelper.RemoveUserByAdminID(subAdminId, adminCtx.ID, models.RoleSubAdmin)
	if err != nil {
		logrus.Printf("Unable to get Sub-Admin: %s", err)
		utils.RespondError(w, http.StatusInternalServerError, err, "Unable to get Sub-Admin")
		return
	}
	logrus.Printf("Sub-Admin remove successfully.")
	utils.RespondJSON(w, http.StatusCreated, models.Message{
		Message: "Sub-Admin remove successfully.",
	})
}
