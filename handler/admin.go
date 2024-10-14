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
		logrus.Errorf("Failed to parse request body: %s", parseErr)
		utils.RespondError(w, http.StatusBadRequest, parseErr, "Failed to parse request body")
		return
	}
	if len(body.Password) < 6 {
		logrus.Errorf("password must be 6 chars long.")
		utils.RespondError(w, http.StatusBadRequest, nil, "password must be 6 chars long")
		return
	}

	if !utils.IsEmailValid(body.Email) {
		logrus.Errorf("Invalid Email.")
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Email.")
		return
	}

	exists, existsErr := dbHelper.IsUserRoleExists(body.Email, models.RoleSubAdmin)
	if existsErr != nil {
		logrus.Errorf("Failed to check user role existence: %s", existsErr)
		utils.RespondError(w, http.StatusConflict, existsErr, "Failed to check Sub-Admin existence")
		return
	}
	if exists {
		logrus.Errorf("Sub-Admin already exists")
		utils.RespondError(w, http.StatusConflict, nil, "Sub-Admin already exists")
		return
	}
	hashedPassword, hasErr := utils.HashPassword(body.Password)
	if hasErr != nil {
		logrus.Errorf("Failed to secure password: %s", hasErr)
		utils.RespondError(w, http.StatusInternalServerError, hasErr, "Failed to secure password")
		return
	}
	userID, existsErr := dbHelper.IsUserExists(body.Email)
	if existsErr != nil {
		logrus.Errorf("Failed to check user existence: %s", existsErr)
		utils.RespondError(w, http.StatusInternalServerError, existsErr, "Failed to check user existence")
		return
	}
	txErr := database.Tx(func(tx *sqlx.Tx) error {
		if len(userID) > 0 {
			roleErr := dbHelper.CreateUserRole(tx, userID, adminCtx.ID, models.RoleSubAdmin)
			if roleErr != nil {
				logrus.Errorf("Failed to create User Role: %s", roleErr)
				return roleErr
			}
		} else {
			userID, saveErr := dbHelper.CreateUser(tx, body.Name, body.Email, hashedPassword)
			if saveErr != nil {
				logrus.Errorf("Failed to Save Sub-Admin: %s", saveErr)
				return saveErr
			}
			roleErr := dbHelper.CreateUserRole(tx, userID, adminCtx.ID, models.RoleSubAdmin)
			if roleErr != nil {
				logrus.Errorf("Failed to create User Role: %s", roleErr)
				return roleErr
			}
		}
		return nil
	})
	if txErr != nil {
		logrus.Errorf("Failed to create SubAdmin: %s", txErr)
		utils.RespondError(w, http.StatusInternalServerError, txErr, "Failed to create user")
		return
	}
	logrus.Infof("SubAdmin Created successfully.")
	utils.RespondJSON(w, http.StatusCreated, models.Message{
		Message: "SubAdmin Created successfully",
	})
}

func GetSubAdmins(w http.ResponseWriter, r *http.Request) {
	Filters := utils.GetFilters(r)
	if Filters.Email != "" && !utils.IsEmailValid(Filters.Email) {
		logrus.Errorf("Invalid Filter Email.")
		utils.RespondError(w, http.StatusNotAcceptable, nil, "Invalid Filter Email.")
		return
	}
	UserCount, countErr := dbHelper.GetUserCount(models.RoleSubAdmin, Filters)
	if countErr != nil {
		logrus.Errorf("Unable to get Users: %s", countErr)
		utils.RespondError(w, http.StatusInternalServerError, countErr, "Unable to get Users")
		return
	}
	subAdmins, err := dbHelper.GetUsers(models.RoleSubAdmin, Filters)
	if err != nil {
		logrus.Errorf("Unable to get Sub-Admin: %s", err)
		utils.RespondError(w, http.StatusInternalServerError, err, "Unable to get Sub-Admin")
		return
	}
	logrus.Infof("Get subAdmin successfully.")
	utils.RespondJSON(w, http.StatusOK, models.GetSubAdmins{
		Message:    "Get subAdmin successfully.",
		SubAdmins:  subAdmins,
		TotalCount: UserCount,
		PageSize:   Filters.PageSize,
		PageNumber: Filters.PageNumber,
	})
}

func RegisterAdmin() {
	adminExist, adminErr := dbHelper.IsAnyRoleExist(models.RoleAdmin)
	if adminErr != nil {
		logrus.Errorf("Any Admin Exist: %s", adminErr)
		return
	}
	if adminExist {
		logrus.Errorf("Any Other Admin Already Exist.")
		return
	}
	exists, existsErr := dbHelper.IsUserRoleExists(os.Getenv("ADMIN_EMAIL"), models.RoleAdmin)
	if existsErr != nil {
		logrus.Errorf("User Exist: %s", existsErr)
		return
	}
	if exists {
		logrus.Errorf("User Exist.")
		return
	}
	hashedPassword, hasErr := utils.HashPassword(os.Getenv("ADMIN_FIRST_PASSWORD"))
	if hasErr != nil {
		logrus.Errorf("Unable to Hash Password: %s", existsErr)
		return
	}
	userID, userExistsErr := dbHelper.IsUserExists(os.Getenv("ADMIN_EMAIL"))
	if userExistsErr != nil {
		logrus.Errorf("User Exist: %s", userExistsErr)
		return
	}
	txErr := database.Tx(func(tx *sqlx.Tx) error {
		if len(userID) > 0 {
			roleErr := dbHelper.CreateUserRole(tx, userID, userID, models.RoleAdmin)
			if roleErr != nil {
				logrus.Errorf("User Role: %s", roleErr)
				return roleErr
			}
		} else {
			userID, saveErr := dbHelper.CreateUser(tx, os.Getenv("ADMIN_NAME"), os.Getenv("ADMIN_EMAIL"), hashedPassword)
			if saveErr != nil {
				logrus.Errorf("User Save: %s", saveErr)
				return saveErr
			}
			roleErr := dbHelper.CreateUserRole(tx, userID, userID, models.RoleAdmin)
			if roleErr != nil {
				logrus.Errorf("User My Role: %s", roleErr)
				return roleErr
			}
		}
		return nil
	})
	if txErr != nil {
		logrus.Infof("Admin Created!")
		return
	}
}

func RemoveSubAdmin(w http.ResponseWriter, r *http.Request) {
	subAdminId := chi.URLParam(r, "subAdminId")
	multipleRoles, rolesErr := dbHelper.UserHaveMultipleRoles(subAdminId)
	if rolesErr != nil {
		logrus.Errorf("Unable to get Users: %s", rolesErr)
		utils.RespondError(w, http.StatusInternalServerError, rolesErr, "Unable to get Users")
		return
	}
	// you have created the transaction but not passing tx in db calls
	txErr := database.Tx(func(tx *sqlx.Tx) error {
		if multipleRoles {
			roleErr := dbHelper.RemoveRole(subAdminId, models.RoleUser)
			if roleErr != nil {
				logrus.Errorf("Failed to Remove Sub-Admin Role: %s", roleErr)
				return roleErr
			}
		} else {
			roleErr := dbHelper.RemoveRole(subAdminId, models.RoleUser)
			if roleErr != nil {
				logrus.Errorf("Failed to Remove Sub-Admin Role: %s", roleErr)
				return roleErr
			}
			userErr := dbHelper.RemoveUser(subAdminId)
			if userErr != nil {
				logrus.Errorf("Failed to remove Sub-Admin: %s", userErr)
				return userErr
			}
		}
		return nil
	})
	if txErr != nil {
		logrus.Errorf("Failed to create user: %s", txErr)
		utils.RespondError(w, http.StatusInternalServerError, txErr, "Failed to create user")
		return
	}
	logrus.Infof("Sub-Admin remove successfully.")
	utils.RespondJSON(w, http.StatusOK, models.Message{
		Message: "Sub-Admin remove successfully.",
	})
}
