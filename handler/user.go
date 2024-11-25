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
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gofrs/uuid"
	log "github.com/sirupsen/logrus"
)

func LoginUser(w http.ResponseWriter, r *http.Request) {

	logrus := log.GetLogger()

	logid, logErr := uuid.NewV4()
	if logErr != nil {
		log.Logger.WithFields(log.Fields{
			"time": time.Now(),
			"uuid": logid,
		}).Error(logErr)
	}

	//TODO :this will made in model and at the time of login role is not taken From the user **DONE**
	var body models.LoginBody

	if parseErr := utils.ParseBody(r.Body, &body); parseErr != nil {
		log.Logger.Errorf("Failed to parse request body: %s", parseErr)
		utils.RespondError(w, http.StatusBadRequest, parseErr, "Failed to parse request body", r.Body)
		return
	}

	config, err := configuration.GetConfig()
	if err != nil {
		log.Logger.WithFields(log.Fields{
			"time":        time.Now(),
			"uuid":        logid.String(),
			"requestBody": body,
		}).Error("Error loading .env file")
	}

	//ToDo please check password here not on repo level when user enter wrong password then it gives status 500 but it will gives 400 **DONE**
	userId, userErr := dbHelper.GetUserIDByPassword(body.Email, body.Password)
	if userErr != nil {
		utils.RespondError(w, http.StatusUnauthorized, userErr, "Failed to find user", body)
		return
	}
	// create user session
	sessionToken, jwtError := utils.JwtToken(userId, config.SessionKey)
	if jwtError != nil {
		utils.RespondError(w, http.StatusInternalServerError, jwtError, jwtError.Error(), body)
		return
	}
	sessionErr := dbHelper.CreateUserSession(database.RMS, userId, sessionToken)
	if sessionErr != nil {
		utils.RespondError(w, http.StatusInternalServerError, sessionErr, "Failed to create user session", body)
		return
	}
	//TODO useErrof instead of printf because we have logging error  not an info level **DONE**
	log.Logger.WithFields(log.Fields{
		"time":        time.Now(),
		"uuid":        logid.String(),
		"requestBody": body,
		"responseBody": models.Login{
			Token:   sessionToken,
			Type:    "Bearer",
			Message: "Login Successfully.",
		},
	}).Info("Login Successfully.")
	utils.RespondJSON(w, http.StatusCreated, models.Login{
		Token:   sessionToken,
		Type:    "Bearer",
		Message: "Login Successfully.",
	})
}

func GetInfo(w http.ResponseWriter, r *http.Request) {

	logrus := log.GetLogger()
	logid, logErr := uuid.NewV4()
	if logErr != nil {
		log.Logger.WithFields(log.Fields{
			"uuid": logid,
		}).Error(logErr)
	}
	userCtx := middlewares.UserContext(r)
	log.logger.WithFields(log.Fields{
		"responseBody": models.GetUser{
			Message: "Get information Successfully.",
			User:    *userCtx,
		},
	}).Info("Get information Successfully.")
	utils.RespondJSON(w, http.StatusOK, models.GetUser{
		Message: "Get information Successfully.",
		User:    *userCtx,
	})
}

func Logout(w http.ResponseWriter, r *http.Request) {

	logrus := log.GetLogger()
	logid, logErr := uuid.NewV4()
	if logErr != nil {
		log.Logger.WithFields(log.Fields{
			"time": time.Now(),
			"uuid": logid,
		}).Error(logErr)
	}
	token := strings.Split(r.Header.Get("authorization"), " ")[1]
	err := dbHelper.DeleteSessionToken(token)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to logout user", "")
		return
	}
	log.Logger.WithFields(log.Fields{
		"time": time.Now(),
		"uuid": logid,
		"responseBody": models.Message{
			Message: "Logout Successfully.",
		},
	}).Info("Logout Successfully.")
	utils.RespondJSON(w, http.StatusAccepted, models.Message{
		Message: "Logout Successfully.",
	})
}

// todo :- use validator package to validate empty string or not in case if any entry of updating is empty then we should return not update the existing details because updating paylload is not valid
func UpdateSelfInfo(w http.ResponseWriter, r *http.Request) {

	logrus := log.GetLogger()
	logid, logErr := uuid.NewV4()
	if logErr != nil {
		log.Logger.WithFields(log.Fields{
			"time": time.Now(),
			"uuid": logid,
		}).Error(logErr)
	}
	var body models.RegisterUserBody

	userContext := middlewares.UserContext(r)
	if parseErr := utils.ParseBody(r.Body, &body); parseErr != nil {
		utils.RespondError(w, http.StatusBadRequest, parseErr, "Failed to parse request body", r.Body)
		return
	}
	if body.Name == "" {
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Name.", body)
		return
	}
	if !utils.IsEmailValid(body.Email) {
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Email.", body)
		return
	}
	if body.Password == "" {
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Password.", body)
		return
	} else {
		hashedPassword, hasErr := utils.HashPassword(body.Password)
		if hasErr != nil {
			utils.RespondError(w, http.StatusInternalServerError, hasErr, "Failed to secure password", body)
			return
		}
		body.Password = hashedPassword
	}
	err := dbHelper.UpdateUserInfo(r.Context(), userContext.ID, body.Name, body.Email, body.Password)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed update User", body)
		return
	}
	log.Logger.WithFields(log.Fields{
		"time":        time.Now(),
		"uuid":        logid.String(),
		"requestBody": body,
		"responseBody": models.Message{
			Message: "User update successfully",
		},
	}).Info("User update successfully")
	utils.RespondJSON(w, http.StatusCreated, models.Message{
		Message: "User update successfully",
	})
}

func AddAddress(w http.ResponseWriter, r *http.Request) {

	logrus := log.GetLogger()
	logid, logErr := uuid.NewV4()
	if logErr != nil {
		log.Logger.WithFields(log.Fields{
			"time": time.Now(),
			"uuid": logid,
		}).Error(logErr)
	}
	var body models.AddUserAddressBody
	userCtx := middlewares.UserContext(r)
	if parseErr := utils.ParseBody(r.Body, &body); parseErr != nil {
		utils.RespondError(w, http.StatusBadRequest, parseErr, "Failed to parse request body", r.Body)
		return
	}

	if len(body.Address) == 0 {
		utils.RespondError(w, http.StatusBadRequest, nil, "Address can't be null.", body)
		return
	}

	if len(body.State) == 0 {
		utils.RespondError(w, http.StatusBadRequest, nil, "State can't be null.", body)
		return
	}

	if len(body.City) == 0 {
		utils.RespondError(w, http.StatusBadRequest, nil, "City can't be null.", body)
		return
	}

	if len(body.PinCode) != 6 {
		utils.RespondError(w, http.StatusBadRequest, nil, "PinCode must 6 digit.", body)
		return
	}

	if body.Lat > 90 || body.Lat < -90 {
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Latitude.", body)
		return
	}

	if body.Lng > 180 || body.Lng < -180 {
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Longitude.", body)
		return
	}
	addressErr := dbHelper.CreateUserAddress(userCtx.ID, body.Address, body.State, body.City, body.PinCode, body.Lat, body.Lng)
	if addressErr != nil {
		utils.RespondError(w, http.StatusInternalServerError, addressErr, "Failed to create Address", body)
		return
	}
	log.Logger.WithFields(log.Fields{
		"time":        time.Now(),
		"uuid":        logid.String(),
		"requestBody": body,
		"responseBody": models.Message{
			Message: "Address Created successfully.",
		},
	}).Info("Address Created successfully")
	utils.RespondJSON(w, http.StatusCreated, models.Message{
		Message: "Address Created successfully.",
	})
}

func UpdateAddress(w http.ResponseWriter, r *http.Request) {

	logrus := log.GetLogger()
	logid, logErr := uuid.NewV4()
	if logErr != nil {
		log.Logger.WithFields(log.Fields{
			"time": time.Now(),
			"uuid": logid,
		}).Error(logErr)
	}
	addressId := chi.URLParam(r, "addressId")
	var body models.AddUserAddressBody

	if parseErr := utils.ParseBody(r.Body, &body); parseErr != nil {
		utils.RespondError(w, http.StatusBadRequest, parseErr, "Failed to parse request body", r.Body)
		return
	}

	if len(body.Address) == 0 {
		utils.RespondError(w, http.StatusBadRequest, nil, "Address can't be null.", body)
		return
	}

	if len(body.State) == 0 {
		utils.RespondError(w, http.StatusBadRequest, nil, "State can't be null.", body)
		return
	}

	if len(body.City) == 0 {
		utils.RespondError(w, http.StatusBadRequest, nil, "City can't be null.", body)
		return
	}

	if len(body.PinCode) != 6 {
		utils.RespondError(w, http.StatusBadRequest, nil, "PinCode must 6 digit.", body)
		return
	}

	if body.Lat > 90 || body.Lat < -90 {
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Latitude.", body)
		return
	}

	if body.Lng > 180 || body.Lng < -180 {
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Longitude.", body)
		return
	}

	err := dbHelper.UpdateUserAddress(addressId, body.Address, body.State, body.City, body.PinCode, body.Lat, body.Lng)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to update Address:", body)
		return
	}
	log.Logger.WithFields(log.Fields{
		"time":        time.Now(),
		"uuid":        logid.String(),
		"requestBody": body,
		"responseBody": models.Message{
			Message: "Address Update successfully",
		},
	}).Info("Address Created successfully")
	utils.RespondJSON(w, http.StatusAccepted, models.Message{
		Message: "Address Update successfully",
	})
}

// Restaurant

func GetRestaurantDistance(w http.ResponseWriter, r *http.Request) {

	logrus := log.GetLogger()
	logid, logErr := uuid.NewV4()
	if logErr != nil {
		log.Logger.WithFields(log.Fields{
			"time": time.Now(),
			"uuid": logid,
		}).Error(logErr)
	}
	restaurantId := r.URL.Query().Get("restaurantId")
	userCtx := middlewares.UserContext(r)
	addressId := r.URL.Query().Get("addressId")

	Restaurant, err := dbHelper.GetRestaurantByID(restaurantId)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Unable to get Restaurant", r.URL.Query())
		return
	}

	userAddress, addressErr := utils.GetUserAddressById(addressId, userCtx.UserAddresses)
	if addressErr != nil {
		utils.RespondError(w, http.StatusBadRequest, nil, "Address not exist", r.URL.Query())
		return
	}

	Distance, Unit := utils.CalculateDistance(userAddress.Lat, userAddress.Lng, Restaurant.Lat, Restaurant.Lng)
	log.Logger.WithFields(log.Fields{
		"time":          time.Now(),
		"uuid":          logid,
		"requestParams": r.URL.Query(),
		"responseBody": models.RestaurantDistance{
			Message:      "Restaurant Distance Calculated successfully.",
			Distance:     Distance,
			DistanceUnit: Unit,
		},
	}).Info("Restaurant Distance Calculated in %s successfully.", Unit)
	utils.RespondJSON(w, http.StatusOK, models.RestaurantDistance{
		Message:      "Restaurant Distance Calculated successfully.",
		Distance:     Distance,
		DistanceUnit: Unit,
	})
}
