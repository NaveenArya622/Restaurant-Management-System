package handler

import (
	"net/http"
	"rms/configuration"
	"rms/database"
	"rms/database/dbHelper"
	"rms/log"
	"rms/middlewares"
	"rms/models"
	"rms/utils"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
	"golang.org/x/sync/errgroup"
)

func RegisterSubAdmin(w http.ResponseWriter, r *http.Request) {
	//logrus := log.GetLogger()
	//logid, logErr := uuid.NewV4()
	//if logErr != nil {
	//	log.Logger.WithFields(log.Fields{
	//		"time": time.Now(),
	//		"uuid": logid,
	//	}).Error(logErr)
	//}
	var body models.RegisterUserBody
	adminCtx := middlewares.UserContext(r)
	if parseErr := utils.ParseBody(r.Body, &body); parseErr != nil {
		utils.RespondError(w, http.StatusBadRequest, parseErr, "Failed to parse request body", body)
		return
	}
	if len(body.Password) < 6 {
		utils.RespondError(w, http.StatusBadRequest, nil, "password must be 6 chars long", body)
		return
	}

	if !utils.IsEmailValid(body.Email) {
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Email.", body)
		return
	}

	exists, existsErr := dbHelper.IsUserRoleExists(body.Email, models.RoleSubAdmin)
	if existsErr != nil {
		utils.RespondError(w, http.StatusConflict, existsErr, "Failed to check Sub-Admin existence", body)
		return
	}
	if exists {
		utils.RespondError(w, http.StatusConflict, nil, "Sub-Admin already exists", body)
		return
	}
	hashedPassword, hasErr := utils.HashPassword(body.Password)
	if hasErr != nil {
		utils.RespondError(w, http.StatusInternalServerError, hasErr, "Failed to secure password", body)
		return
	}
	userID, existsErr := dbHelper.IsUserExists(body.Email)
	if existsErr != nil {
		utils.RespondError(w, http.StatusInternalServerError, existsErr, "Failed to check user existence", body)
		return
	}
	txErr := database.Tx(func(tx *sqlx.Tx) error {
		if len(userID) > 0 {
			roleErr := dbHelper.CreateUserRole(tx, userID, adminCtx.ID, models.RoleSubAdmin)
			if roleErr != nil {
				log.Logger.Errorf("Failed to create User Role: %s", roleErr)
				return roleErr
			}
		} else {
			userID, saveErr := dbHelper.CreateUser(tx, body.Name, body.Email, hashedPassword)
			if saveErr != nil {
				log.Logger.Errorf("Failed to Save Sub-Admin: %s", saveErr)
				return saveErr
			}
			roleErr := dbHelper.CreateUserRole(tx, userID, adminCtx.ID, models.RoleSubAdmin)
			if roleErr != nil {
				log.Logger.Errorf("Failed to create User Role: %s", roleErr)
				return roleErr
			}
		}
		return nil
	})
	if txErr != nil {
		log.Logger.Errorf("Failed to create SubAdmin: %s", txErr)
		utils.RespondError(w, http.StatusInternalServerError, txErr, "Failed to create user", body)
		return
	}
	log.Logger.WithFields(log.Fields{
		"time":        time.Now(),
		"uuid":        logid.String(),
		"requestBody": body,
		"responseBody": models.Message{
			Message: "SubAdmin Created successfully",
		},
	}).Info("SubAdmin Created successfully.")
	utils.RespondJSON(w, http.StatusCreated, models.Message{
		Message: "SubAdmin Created successfully",
	})
}

func GetSubAdmins(w http.ResponseWriter, r *http.Request) {
	//logrus := log.GetLogger()
	//logid, logErr := uuid.NewV4()
	//if logErr != nil {
	//	log.Logger.WithFields(log.Fields{
	//		"time": time.Now(),
	//		"uuid": logid,
	//	}).Error(logErr)
	//}
	Filters := utils.GetFilters(r)
	if Filters.Email != "" && !utils.IsEmailValid(Filters.Email) {
		utils.RespondError(w, http.StatusNotAcceptable, nil, "Invalid Filter Email.", Filters)
		return
	}
	var subAdminsCount int64
	subAdmins := make([]models.User, 0)
	var errGroup errgroup.Group
	errGroup.Go(func() error {
		var err error
		subAdminsCount, err = dbHelper.GetUserCount(models.RoleSubAdmin, Filters)
		if err != nil {
			log.Logger.Errorf("Unable to get Users Sub-Admin: %s", err)
		}
		return err
	})
	errGroup.Go(func() error {
		var err error
		subAdmins, err = dbHelper.GetUsers(models.RoleSubAdmin, Filters)
		if err != nil {
			log.Logger.Errorf("Unable to get Sub-Admin: %s", err)
		}
		return err
	})
	if err := errGroup.Wait(); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Unable to get subAdmin", Filters)
		return
	}

	utils.RespondJSON(w, http.StatusOK, models.GetSubAdmins{
		Message:    "Get subAdmin successfully.",
		SubAdmins:  subAdmins,
		TotalCount: subAdminsCount,
		PageSize:   Filters.PageSize,
		PageNumber: Filters.PageNumber,
	})
}

func RegisterAdmin(config configuration.Config) {
	adminExist, adminErr := dbHelper.IsAnyRoleExist(models.RoleAdmin)
	if adminErr != nil {
		log.Logger.WithFields(log.Fields{
			"time":        time.Now(),
			"uuid":        logid.String(),
			"requestBody": config,
		}).Error("Any Admin Exist: %s", adminErr)
		return
	}
	if adminExist {
		log.Logger.WithFields(log.Fields{
			"time":        time.Now(),
			"uuid":        logid.String(),
			"requestBody": config,
		}).Error("Any Other Admin Already Exist.")
		return
	}
	exists, existsErr := dbHelper.IsUserRoleExists(config.AdminEmail, models.RoleAdmin)
	if existsErr != nil {
		log.Logger.WithFields(log.Fields{
			"time":        time.Now(),
			"uuid":        logid.String(),
			"requestBody": config,
		}).Error("User Exist: %s", existsErr)
		return
	}
	if exists {
		log.Logger.WithFields(log.Fields{
			"time":        time.Now(),
			"uuid":        logid.String(),
			"requestBody": config,
		}).Error("User Exist.")
		return
	}
	hashedPassword, hasErr := utils.HashPassword(config.AdminFirstPassword)
	if hasErr != nil {
		log.Logger.WithFields(log.Fields{
			"time":        time.Now(),
			"uuid":        logid.String(),
			"requestBody": config,
		}).Error("Unable to Hash Password: %s", existsErr)
		return
	}
	userID, userExistsErr := dbHelper.IsUserExists(config.AdminEmail)
	if userExistsErr != nil {
		log.Logger.WithFields(log.Fields{
			"time":        time.Now(),
			"uuid":        logid.String(),
			"requestBody": config,
		}).Error("User Exist: %s", userExistsErr)
		return
	}
	txErr := database.Tx(func(tx *sqlx.Tx) error {
		if len(userID) > 0 {
			roleErr := dbHelper.CreateUserRole(tx, userID, userID, models.RoleAdmin)
			if roleErr != nil {
				log.Logger.WithFields(log.Fields{
					"time":        time.Now(),
					"uuid":        logid.String(),
					"requestBody": config,
				}).Error("User Role: %s", roleErr)
				return roleErr
			}
		} else {
			userID, saveErr := dbHelper.CreateUser(tx, config.AdminName, config.AdminEmail, hashedPassword)
			if saveErr != nil {
				log.Logger.WithFields(log.Fields{
					"time":        time.Now(),
					"uuid":        logid.String(),
					"requestBody": config,
				}).Error("User Save: %s", saveErr)
				return saveErr
			}
			roleErr := dbHelper.CreateUserRole(tx, userID, userID, models.RoleAdmin)
			if roleErr != nil {
				log.Logger.WithFields(log.Fields{
					"time":        time.Now(),
					"uuid":        logid.String(),
					"requestBody": config,
				}).Error("User My Role: %s", roleErr)
				return roleErr
			}
		}
		return nil
	})
	if txErr != nil {
		log.Logger.WithFields(log.Fields{
			"time":        time.Now(),
			"uuid":        logid.String(),
			"requestBody": config,
		}).Error("Unable to Create Admin.")
		return
	}
	log.Logger.WithFields(log.Fields{
		"time":        time.Now(),
		"uuid":        logid.String(),
		"requestBody": config,
	}).Info("Admin Created!")
	return

}

func RemoveSubAdmin(w http.ResponseWriter, r *http.Request) {
	logrus := log.GetLogger()
	logid, logErr := uuid.NewV4()
	if logErr != nil {
		log.Logger.WithFields(log.Fields{
			"time": time.Now(),
			"uuid": logid,
		}).Error(logErr)
	}
	subAdminId := chi.URLParam(r, "subAdminId")
	multipleRoles, rolesErr := dbHelper.UserHaveMultipleRoles(subAdminId)
	if rolesErr != nil {
		utils.RespondError(w, http.StatusInternalServerError, rolesErr, "Unable to get Users", "subAdminId: "+subAdminId)
		return
	}
	txErr := database.Tx(func(tx *sqlx.Tx) error {
		if multipleRoles {
			roleErr := dbHelper.RemoveRole(tx, subAdminId, models.RoleUser)
			if roleErr != nil {
				return roleErr
			}
		} else {
			roleErr := dbHelper.RemoveRole(tx, subAdminId, models.RoleUser)
			if roleErr != nil {
				return roleErr
			}
			userErr := dbHelper.RemoveUser(tx, subAdminId)
			if userErr != nil {
				return userErr
			}
		}
		return nil
	})
	if txErr != nil {
		utils.RespondError(w, http.StatusInternalServerError, txErr, "Failed to create user", "subAdminId: "+subAdminId)
		return
	}
	log.Logger.WithFields(log.Fields{
		"time":        time.Now(),
		"uuid":        logid.String(),
		"requestBody": "subAdminId: " + subAdminId,
		"responseBody": models.Message{
			Message: "SubAdmin Created successfully",
		},
	}).Info("Sub-Admin remove successfully.")
	utils.RespondJSON(w, http.StatusOK, models.Message{
		Message: "Sub-Admin remove successfully.",
	})
}
