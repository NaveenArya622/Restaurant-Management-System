package handler

import (
	"net/http"
	"rms/database"
	"rms/database/dbHelper"
	"rms/middlewares"
	"rms/models"
	"rms/utils"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

func LoginUser(w http.ResponseWriter, r *http.Request) {
	//TODO :this will made in model and at the time of login role is not taken From the user **DONE**
	var body models.LoginBody

	if parseErr := utils.ParseBody(r.Body, &body); parseErr != nil {
		logrus.Errorf("Failed to parse request body: %s", parseErr)
		utils.RespondError(w, http.StatusBadRequest, parseErr, "Failed to parse request body")
		return
	}
	//ToDo please check password here not on repo level when user enter wrong password then it gives status 500 but it will gives 400 **DONE**
	userId, userRoleId, userErr := dbHelper.GetUserRoleIDByPassword(body.Email, body.Password, body.Role)
	if userErr != nil {
		logrus.Errorf("Failed to find user: %s", userErr)
		utils.RespondError(w, http.StatusUnauthorized, userErr, "Failed to find user")
		return
	}
	// create user session
	sessionToken, jwtError := utils.JwtToken(userId, userRoleId)
	if jwtError != nil {
		logrus.Errorf(jwtError.Error())
		utils.RespondError(w, http.StatusInternalServerError, jwtError, jwtError.Error())
		return
	}
	sessionErr := dbHelper.CreateUserSession(database.RMS, userId, userRoleId, sessionToken)
	if sessionErr != nil {
		logrus.Errorf("Failed to create user session: %s", sessionErr)
		utils.RespondError(w, http.StatusInternalServerError, sessionErr, "Failed to create user session")
		return
	}
	//TODO useErrorf instead of printf because we have logging error  not an info level **DONE**
	logrus.Infof("Login Successfully.")
	utils.RespondJSON(w, http.StatusCreated, models.Login{
		Token:   sessionToken,
		Type:    "Bearer",
		Message: "Login Successfully.",
	})
}

func GetInfo(w http.ResponseWriter, r *http.Request) {
	userCtx := middlewares.UserContext(r)
	logrus.Infof("Get information Successfully.")
	utils.RespondJSON(w, http.StatusOK, models.GetUser{
		Message: "Get information Successfully.",
		User:    *userCtx,
	})
}

func Logout(w http.ResponseWriter, r *http.Request) {
	token := strings.Split(r.Header.Get("authorization"), " ")[1]
	err := dbHelper.DeleteSessionToken(token)
	if err != nil {
		logrus.Errorf("Failed to logout user: %s", err)
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to logout user")
		return
	}
	logrus.Infof("Logout Successfully.")
	utils.RespondJSON(w, http.StatusAccepted, models.Message{
		Message: "Logout Successfully.",
	})
}

// TODO :- use validator package to validate empty string or not in case if any entry of updating is empty then we should return not update the existing details because updating paylload is not valid(try this use lte and get for consstraint)
func UpdateSelfInfo(w http.ResponseWriter, r *http.Request) {
	var body models.RegisterUserBody

	adminCtx := middlewares.UserContext(r)
	if parseErr := utils.ParseBody(r.Body, &body); parseErr != nil {
		logrus.Errorf("Failed to parse request body: %s", parseErr)
		utils.RespondError(w, http.StatusBadRequest, parseErr, "Failed to parse request body")
		return
	}
	if body.Name == "" {
		body.Name = adminCtx.Name
	}
	if !utils.IsEmailValid(body.Email) {
		if body.Email != "" {
			logrus.Errorf("Invalid Email.")
			utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Email.")
			return
		}
		body.Email = adminCtx.Email
	}
	if body.Password == "" {
		body.Password = adminCtx.Password
	} else {
		hashedPassword, hasErr := utils.HashPassword(body.Password)
		if hasErr != nil {
			logrus.Errorf("Failed to secure password: %s", hasErr)
			utils.RespondError(w, http.StatusInternalServerError, hasErr, "Failed to secure password")
			return
		}
		body.Password = hashedPassword
	}
	err := dbHelper.UpdateUserInfo(adminCtx.ID, body.Name, body.Email, body.Password)
	if err != nil {
		logrus.Errorf("Failed update User: %s", err)
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed update User")
		return
	}
	logrus.Infof("User update successfully")
	utils.RespondJSON(w, http.StatusCreated, models.Message{
		Message: "User update successfully",
	})
}

func AddAddress(w http.ResponseWriter, r *http.Request) {
	var body models.AddUserAddressBody
	userCtx := middlewares.UserContext(r)
	if parseErr := utils.ParseBody(r.Body, &body); parseErr != nil {
		logrus.Errorf("Failed to parse request body: %s", parseErr)
		utils.RespondError(w, http.StatusBadRequest, parseErr, "Failed to parse request body")
		return
	}

	if len(body.Address) > 30 || len(body.Address) <= 2 {
		logrus.Errorf("Address must be with in 2 to 30 letter.")
		utils.RespondError(w, http.StatusBadRequest, nil, "Address must be with in 2 to 30 letter.")
		return
	}

	if len(body.State) > 16 || len(body.State) <= 2 {
		logrus.Errorf("State must be with in 2 to 16 letter.")
		utils.RespondError(w, http.StatusBadRequest, nil, "State must be with in 2 to 16 letter.")
		return
	}

	if len(body.City) > 20 || len(body.City) <= 2 {
		logrus.Errorf("City must be with in 2 to 20 letter.")
		utils.RespondError(w, http.StatusBadRequest, nil, "City must be with in 2 to 20 letter.")
		return
	}

	if len(body.PinCode) != 6 {
		logrus.Errorf("PinCode must 6 digit.")
		utils.RespondError(w, http.StatusBadRequest, nil, "PinCode must 6 digit.")
		return
	}

	if body.Lat > 90 || body.Lat < -90 {
		logrus.Errorf("Invalid Latitude.")
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Latitude.")
		return
	}

	if body.Lng > 180 || body.Lng < -180 {
		logrus.Errorf("Invalid Longitude.")
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Longitude.")
		return
	}
	addressErr := dbHelper.CreateUserAddress(userCtx.ID, body.Address, body.State, body.City, body.PinCode, body.Lat, body.Lng)
	if addressErr != nil {
		logrus.Errorf("Failed to create Address: %s", addressErr)
		utils.RespondError(w, http.StatusInternalServerError, addressErr, "Failed to create Address")
		return
	}
	logrus.Infof("Address Created successfully")
	utils.RespondJSON(w, http.StatusCreated, models.Message{
		Message: "Address Created successfully.",
	})
}

func UpdateAddress(w http.ResponseWriter, r *http.Request) {
	addressId := chi.URLParam(r, "addressId")
	var body models.AddUserAddressBody

	userCtx := middlewares.UserContext(r)
	if parseErr := utils.ParseBody(r.Body, &body); parseErr != nil {
		logrus.Errorf("Failed to parse request body: %s", parseErr)
		utils.RespondError(w, http.StatusBadRequest, parseErr, "Failed to parse request body")
		return
	}

	userAddress, addressErr := utils.GetUserAddressById(addressId, userCtx.UserAddresses)
	if addressErr != nil {
		logrus.Errorf("Address not exist: %s", addressErr)
		utils.RespondError(w, http.StatusBadRequest, nil, "Address not exist")
		return
	}

	if len(body.Address) > 30 || len(body.Address) <= 2 {
		if body.Address != "" {
			logrus.Errorf("Address must be with in 2 to 30 letter.")
			utils.RespondError(w, http.StatusBadRequest, nil, "Address must be with in 2 to 30 letter.")
			return
		}
		body.Address = userAddress.Address
	}

	if len(body.State) > 16 || len(body.State) <= 2 {
		if body.State != "" {
			logrus.Errorf("State must be with in 2 to 16 letter.")
			utils.RespondError(w, http.StatusBadRequest, nil, "State must be with in 2 to 16 letter.")
			return
		}
		body.State = userAddress.State
	}

	if len(body.City) > 20 || len(body.City) <= 2 {
		if body.City != "" {
			logrus.Errorf("City must be with in 2 to 20 letter.")
			utils.RespondError(w, http.StatusBadRequest, nil, "City must be with in 2 to 20 letter.")
			return
		}
		body.City = userAddress.City
	}

	if len(body.PinCode) != 6 {
		if body.PinCode != "" {
			logrus.Errorf("PinCode must 6 digit.")
			utils.RespondError(w, http.StatusBadRequest, nil, "PinCode must 6 digit.")
			return
		}
		body.PinCode = userAddress.PinCode
	}

	if body.Lat > 90 || body.Lat < -90 {
		if body.Lat != 0 {
			logrus.Errorf("Invalid Latitude.")
			utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Latitude.")
			return
		}
		body.Lat = userAddress.Lat
	}

	if body.Lng > 180 || body.Lng < -180 {
		if body.Lat != 0 {
			logrus.Errorf("Invalid Longitude.")
			utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Longitude.")
			return
		}
		body.Lng = userAddress.Lng
	}

	err := dbHelper.UpdateUserAddress(addressId, body.Address, body.State, body.City, body.PinCode, body.Lat, body.Lng)
	if err != nil {
		logrus.Errorf("Failed to update Address: %s", addressErr)
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to update Address:")
		return
	}
	logrus.Infof("Address Created successfully")
	utils.RespondJSON(w, http.StatusAccepted, models.Message{
		Message: "Address Update successfully",
	})
}

// Restaurant

func GetRestaurantDistance(w http.ResponseWriter, r *http.Request) {
	restaurantId := r.URL.Query().Get("restaurantId")
	userCtx := middlewares.UserContext(r)
	addressId := r.URL.Query().Get("addressId")

	Restaurant, err := dbHelper.GetRestaurantByID(restaurantId)
	if err != nil {
		logrus.Errorf("Unable to get Restaurant: %s", err)
		utils.RespondError(w, http.StatusInternalServerError, err, "Unable to get Restaurant")
		return
	}

	userAddress, addressErr := utils.GetUserAddressById(addressId, userCtx.UserAddresses)
	if addressErr != nil {
		logrus.Errorf("Address not exist: %s", addressErr)
		utils.RespondError(w, http.StatusBadRequest, nil, "Address not exist")
		return
	}

	Distance, Unit := utils.CalculateDistance(userAddress.Lat, userAddress.Lng, Restaurant.Lat, Restaurant.Lng)
	logrus.Infof("Restaurant Distance Calculated in %s successfully.", Unit)
	utils.RespondJSON(w, http.StatusOK, models.RestaurantDistance{
		Message:      "Restaurant Distance Calculated successfully.",
		Distance:     Distance,
		DistanceUnit: Unit,
	})
}
