package handler

import (
	"net/http"
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
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

func RegisterUser(w http.ResponseWriter, r *http.Request) {
	logrus := log.GetLogger()
	logid, logErr := uuid.NewV4()
	if logErr != nil {
		log.Logger.WithFields(log.Fields{
			"time": time.Now(),
			"uuid": logid,
		}).Error(logErr)
	}
	var body models.RegisterUserBody
	subAdminCtx := middlewares.UserContext(r)
	if parseErr := utils.ParseBody(r.Body, &body); parseErr != nil {
		utils.RespondError(w, http.StatusBadRequest, parseErr, "Failed to parse request body", r.Body)
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

	exists, existsErr := dbHelper.IsUserRoleExists(body.Email, models.RoleUser)
	if existsErr != nil {
		utils.RespondError(w, http.StatusInternalServerError, existsErr, "Failed to check user role existence", body)
		return
	}
	if exists {
		utils.RespondError(w, http.StatusBadRequest, nil, "user already exists", body)
		return
	}
	hashedPassword, hasErr := utils.HashPassword(body.Password)
	if hasErr != nil {
		utils.RespondError(w, http.StatusInternalServerError, hasErr, "Failed to secure password", body)
		return
	}
	//todo  use this db call outside transaction **DONE**
	userID, existsErr := dbHelper.IsUserExists(body.Email)
	if existsErr != nil {
		utils.RespondError(w, http.StatusInternalServerError, existsErr, "Failed to check user existence", body)
		return
	}
	txErr := database.Tx(func(tx *sqlx.Tx) error {
		if len(userID) > 0 {
			roleErr := dbHelper.CreateUserRole(tx, userID, subAdminCtx.ID, models.RoleUser)
			if roleErr != nil {
				log.Logger.Errorf("Failed to create User Role: %s", roleErr)
				return roleErr
			}
		} else {
			userID, saveErr := dbHelper.CreateUser(tx, body.Name, body.Email, hashedPassword)
			if saveErr != nil {
				log.Logger.Errorf("Failed to create User: %s", saveErr)
				return saveErr
			}
			roleErr := dbHelper.CreateUserRole(tx, userID, subAdminCtx.ID, models.RoleUser)
			if roleErr != nil {
				log.Logger.Errorf("Failed to failed to create User Role: %s", roleErr)
				return roleErr
			}
		}
		return nil
	})
	if txErr != nil {
		utils.RespondError(w, http.StatusInternalServerError, txErr, "Failed to create user", body)
		return
	}
	log.Logger.WithFields(log.Fields{
		"time":        time.Now(),
		"uuid":        logid.String(),
		"requestBody": body,
		"responseBody": models.Message{
			Message: "user Created successfully",
		},
	}).Info("user Created successfully.")
	utils.RespondJSON(w, http.StatusCreated, models.Message{
		Message: "user Created successfully",
	})
}

func GetUsers(w http.ResponseWriter, r *http.Request) {
	logrus := log.GetLogger()
	logid, logErr := uuid.NewV4()
	if logErr != nil {
		log.Logger.WithFields(log.Fields{
			"time": time.Now(),
			"uuid": logid,
		}).Error(logErr)
	}
	Filters := utils.GetFilters(r)
	if Filters.Email != "" && !utils.IsEmailValid(Filters.Email) {
		log.Logger.Errorf("Invalid Filter Email.")
		utils.RespondError(w, http.StatusExpectationFailed, nil, "Invalid Filter Email.", Filters)
		return
	}
	adminCtx := middlewares.UserContext(r)
	var userCount int64
	users := make([]models.User, 0)
	var errGroup errgroup.Group
	if adminCtx.CurrentRole == models.RoleAdmin {
		errGroup.Go(func() error {
			var err error
			userCount, err = dbHelper.GetUserCount(models.RoleUser, Filters)
			if err != nil {
				log.Logger.Errorf("Unable to get Users Count: %s", err)
			}
			return err
		})
		errGroup.Go(func() error {
			var err error
			users, err = dbHelper.GetUsers(models.RoleUser, Filters)
			if err != nil {
				log.Logger.Errorf("Unable to get Users: %s", err)
			}
			return err
		})
	} else {
		errGroup.Go(func() error {
			var err error
			userCount, err = dbHelper.GetUserCountByAdminID(adminCtx.ID, models.RoleUser, Filters)
			if err != nil {
				log.Logger.Errorf("Unable to get Users Count: %s", err)
			}
			return err
		})
		errGroup.Go(func() error {
			var err error
			users, err = dbHelper.GetUsersByAdminID(adminCtx.ID, models.RoleUser, Filters)
			if err != nil {
				log.Logger.Errorf("Unable to get Users: %s", err)
			}
			return err
		})
	}
	if err := errGroup.Wait(); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Unable to get Users", Filters)
		return
	}
	log.Logger.WithFields(log.Fields{
		"time":        time.Now(),
		"uuid":        logid.String(),
		"requestBody": Filters,
		"responseBody": models.GetUsers{
			Message:    "Get users successfully.",
			Users:      users,
			TotalCount: userCount,
			PageSize:   Filters.PageSize,
			PageNumber: Filters.PageNumber,
		},
	}).Info("Get users successfully.")
	utils.RespondJSON(w, http.StatusOK, models.GetUsers{
		Message:    "Get users successfully.",
		Users:      users,
		TotalCount: userCount,
		PageSize:   Filters.PageSize,
		PageNumber: Filters.PageNumber,
	})
}

// todo :- remove user details also **DONE**
func RemoveUser(w http.ResponseWriter, r *http.Request) {
	logrus := log.GetLogger()
	logid, logErr := uuid.NewV4()
	if logErr != nil {
		log.Logger.WithFields(log.Fields{
			"time": time.Now(),
			"uuid": logid,
		}).Error(logErr)
	}
	//todo :- remove user details **DONE**
	id := chi.URLParam(r, "userId")
	adminCtx := middlewares.UserContext(r)
	multipleRoles, rolesErr := dbHelper.UserHaveMultipleRoles(id)
	if rolesErr != nil {
		log.Logger.Errorf("Unable to get Users: %s", rolesErr)
		utils.RespondError(w, http.StatusInternalServerError, rolesErr, "Unable to get Users", "UID: "+id)
		return
	}
	if adminCtx.CurrentRole == models.RoleAdmin {
		txErr := database.Tx(func(tx *sqlx.Tx) error {
			if multipleRoles {
				roleErr := dbHelper.RemoveRole(tx, id, models.RoleUser)
				if roleErr != nil {
					log.Logger.Errorf("Failed to Remove User Role: %s", roleErr)
					return roleErr
				}
			} else {
				roleErr := dbHelper.RemoveRole(tx, id, models.RoleUser)
				if roleErr != nil {
					log.Logger.Errorf("Failed to Remove User Role: %s", roleErr)
					return roleErr
				}
				userErr := dbHelper.RemoveUser(tx, id)
				if userErr != nil {
					log.Logger.Errorf("Failed to remove User: %s", userErr)
					return userErr
				}
			}
			return nil
		})
		if txErr != nil {
			log.Logger.Errorf("Failed to create user: %s", txErr)
			utils.RespondError(w, http.StatusInternalServerError, txErr, "Failed to create user", "UID: "+id)
			return
		}
	} else {
		txErr := database.Tx(func(tx *sqlx.Tx) error {
			if multipleRoles {
				saveErr := dbHelper.RemoveRoleByAdminID(tx, id, adminCtx.ID, models.RoleUser)
				if saveErr != nil {
					log.Logger.Errorf("Failed to parse request body: %s", saveErr)
					return saveErr
				}
			} else {
				saveErr := dbHelper.RemoveRoleByAdminID(tx, id, adminCtx.ID, models.RoleUser)
				if saveErr != nil {
					log.Logger.Errorf("Failed to parse request body: %s", saveErr)
					return saveErr
				}
				roleErr := dbHelper.RemoveUser(tx, id)
				if roleErr != nil {
					log.Logger.Errorf("Failed to parse request body: %s", roleErr)
					return roleErr
				}
			}
			return nil
		})
		if txErr != nil {
			log.Logger.Errorf("Failed to create user: %s", txErr)
			utils.RespondError(w, http.StatusInternalServerError, txErr, "Failed to create user", "UID: "+id)
			return
		}
	}
	log.Logger.WithFields(log.Fields{
		"time":         time.Now(),
		"uuid":         logid,
		"requestBBody": "UID: " + id,
		"responseBody": models.Message{
			Message: "User removed successfully.",
		},
	}).Info("User removed successfully.")
	utils.RespondJSON(w, http.StatusOK, models.Message{
		Message: "User removed successfully.",
	})
}

// Restaurant

func OpenRestaurant(w http.ResponseWriter, r *http.Request) {
	logid, logErr := uuid.NewV4()
	if logErr != nil {
		log.logger.WithFields(log.Fields{
			"time": time.Now(),
			"uuid": logid,
		}).Error(logErr)
	}
	var body models.OpenRestaurantBody
	adminCtx := middlewares.UserContext(r)
	if parseErr := utils.ParseBody(r.Body, &body); parseErr != nil {
		utils.RespondError(w, http.StatusBadRequest, parseErr, "Failed to parse request body", r.Body)
		return
	}

	if !utils.IsEmailValid(body.Email) {
		utils.RespondError(w, http.StatusExpectationFailed, nil, "Invalid Email.", body)
		return
	}

	exists, existsErr := dbHelper.IsRestaurantExists(body.Email)
	if existsErr != nil {
		utils.RespondError(w, http.StatusInternalServerError, existsErr, "Failed to check Restaurant existence", body)
		return
	}

	if exists {
		utils.RespondError(w, http.StatusAlreadyReported, nil, "Restaurant already exists", body)
		return
	}

	if body.Name == "" {
		utils.RespondError(w, http.StatusExpectationFailed, nil, "Invalid Name", body)
		return
	}
	if !utils.IsEmailValid(body.Email) {
		utils.RespondError(w, http.StatusExpectationFailed, nil, "Invalid Email", body)
		return
	}

	if len(body.Address) == 0 {
		utils.RespondError(w, http.StatusExpectationFailed, nil, "Address can't be null.", body)
		return
	}

	if len(body.State) == 0 {
		utils.RespondError(w, http.StatusExpectationFailed, nil, "State can't be null.", body)
		return
	}

	if len(body.City) == 0 {
		utils.RespondError(w, http.StatusExpectationFailed, nil, "City can't be null.", body)
		return
	}

	if len(body.PinCode) != 6 {
		utils.RespondError(w, http.StatusExpectationFailed, nil, "PinCode must 6 digit.", body)
		return
	}

	if body.Lat > 90 || body.Lat < -90 {
		utils.RespondError(w, http.StatusExpectationFailed, nil, "Invalid Latitude.", body)
		return
	}

	if body.Lng > 180 || body.Lng < -180 {
		utils.RespondError(w, http.StatusExpectationFailed, nil, "Invalid Longitude.", body)
		return
	}
	_, saveErr := dbHelper.CreateRestaurant(body.Name, body.Email, adminCtx.ID, body.Address, body.State, body.City, body.PinCode, body.Lat, body.Lng)
	if saveErr != nil {
		utils.RespondError(w, http.StatusInternalServerError, saveErr, "Failed to open Restaurant", body)
		return
	}
	log.Logger.WithFields(log.Fields{
		"time":        time.Now(),
		"uuid":        logid.String(),
		"requestBody": body,
		"responseBody": models.Message{
			Message: "Restaurant Opened successfully",
		},
	}).Info("Restaurant Opened successfully")
	utils.RespondJSON(w, http.StatusOK, models.Message{
		Message: "Restaurant Opened successfully",
	})
}

func CloseRestaurant(w http.ResponseWriter, r *http.Request) {
	logrus := log.GetLogger()
	logid, logErr := uuid.NewV4()
	if logErr != nil {
		log.Logger.WithFields(log.Fields{
			"time": time.Now(),
			"uuid": logid,
		}).Error(logErr)
	}
	id := chi.URLParam(r, "restaurantId")
	adminCtx := middlewares.UserContext(r)
	var err error
	if adminCtx.CurrentRole == models.RoleAdmin {
		err = dbHelper.CloseRestaurant(id)
	} else {
		err = dbHelper.CloseMyRestaurant(id, adminCtx.ID)
	}
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Unable to get Restaurant", "restaurantId: "+id)
		return
	}
	log.Logger.WithFields(log.Fields{
		"time": time.Now(),
		"uuid": logid,
		"responseBody": models.Message{
			Message: "Restaurant Closed successfully.",
		},
	}).Info("Restaurant Closed successfully.")
	utils.RespondJSON(w, http.StatusOK, models.Message{
		Message: "Restaurant Closed successfully.",
	})
}

func GetRestaurants(w http.ResponseWriter, r *http.Request) {
	logrus := log.GetLogger()
	logid, logErr := uuid.NewV4()
	if logErr != nil {
		log.Logger.WithFields(log.Fields{
			"time": time.Now(),
			"uuid": logid,
		}).Error(logErr)
	}
	Filters := utils.GetFilters(r)
	if Filters.Email != "" && !utils.IsEmailValid(Filters.Email) {
		log.Logger.Errorf("Invalid Restaurants Filter Email.")
		utils.RespondError(w, http.StatusInternalServerError, nil, "Invalid Restaurants Filter Email.", Filters)
		return
	}
	adminCtx := middlewares.UserContext(r)
	//TODO use errgroup when mutiple db calls for less time consuming **DONE**
	var RestaurantsCount int64
	var Restaurants []models.Restaurant
	var errGroup errgroup.Group
	if adminCtx.CurrentRole == models.RoleAdmin || adminCtx.CurrentRole == models.RoleUser {
		errGroup.Go(func() error {
			var err error
			RestaurantsCount, err = dbHelper.GetRestaurantsCount(Filters)
			log.Logger.Errorf("Unable to get Restaurants Count: %s", err)
			return err
		})
		errGroup.Go(func() error {
			var err error
			Restaurants, err = dbHelper.GetRestaurants(Filters)
			log.Logger.Errorf("Unable to get Restaurants: %s", err)
			return err
		})
	} else {
		errGroup.Go(func() error {
			var err error
			RestaurantsCount, err = dbHelper.GetRestaurantsCountByUserID(adminCtx.ID, Filters)
			log.Logger.Errorf("Unable to get Restaurants Count: %s", err)
			return err
		})
		errGroup.Go(func() error {
			var err error
			Restaurants, err = dbHelper.GetRestaurantsByUserID(adminCtx.ID, Filters)
			log.Logger.Errorf("Unable to get Restaurants: %s", err)
			return err
		})
	}
	if err := errGroup.Wait(); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Unable to get Restaurants", Filters)
		return
	}
	//TODO  write this  message below else one time **DONE**
	log.Logger.WithFields(log.Fields{
		"time":        time.Now(),
		"uuid":        logid.String(),
		"requestBody": Filters,
		"responseBody": models.GetRestaurants{
			Message:     "Get Restaurants successfully.",
			Restaurants: Restaurants,
			TotalCount:  RestaurantsCount,
			PageNumber:  Filters.PageNumber,
			PageSize:    Filters.PageSize,
		},
	}).Info("Get Restaurants successfully.")
	utils.RespondJSON(w, http.StatusCreated, models.GetRestaurants{
		Message:     "Get Restaurants successfully.",
		Restaurants: Restaurants,
		TotalCount:  RestaurantsCount,
		PageNumber:  Filters.PageNumber,
		PageSize:    Filters.PageSize,
	})
}

func UpdateRestaurant(w http.ResponseWriter, r *http.Request) {
	logrus := log.GetLogger()
	logid, logErr := uuid.NewV4()
	if logErr != nil {
		log.Logger.WithFields(log.Fields{
			"time": time.Now(),
			"uuid": logid,
		}).Error(logErr)
	}
	restaurantId := chi.URLParam(r, "restaurantId")
	var body models.OpenRestaurantBody

	adminCtx := middlewares.UserContext(r)
	if parseErr := utils.ParseBody(r.Body, &body); parseErr != nil {
		utils.RespondError(w, http.StatusBadRequest, parseErr, "Failed to parse request body", body)
		return
	}

	restaurant, restaurantErr := dbHelper.GetRestaurantByID(restaurantId)
	if restaurantErr != nil {
		utils.RespondError(w, http.StatusBadRequest, restaurantErr, "Restaurant not exist", body)
		return
	}

	if adminCtx.CurrentRole != models.RoleAdmin && restaurant.CreatedBy != adminCtx.ID {
		utils.RespondError(w, http.StatusBadRequest, nil, "Restaurant not Created by: "+string(adminCtx.CurrentRole), body)
		return
	}

	if body.Name == "" {
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Restaurant Name", body)
		return
	}
	if !utils.IsEmailValid(body.Email) {
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Restaurant Email", body)
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

	err := dbHelper.UpdateRestaurant(restaurantId, body.Name, body.Email, body.Address, body.State, body.City, body.PinCode, body.Lat, body.Lng)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to update Restaurant", body)
		return
	}
	log.Logger.WithFields(log.Fields{
		"time":        time.Now(),
		"uuid":        logid.String(),
		"requestBody": body,
		"responseBody": models.Message{
			Message: "Restaurant Updated successfully.",
		},
	}).Info("Restaurant Opened successfully.")
	utils.RespondJSON(w, http.StatusCreated, models.Message{
		Message: "Restaurant Updated successfully.",
	})
}

// Restaurant Dishes

func AddRestaurantDish(w http.ResponseWriter, r *http.Request) {
	logrus := log.GetLogger()
	logid, logErr := uuid.NewV4()
	if logErr != nil {
		log.Logger.WithFields(log.Fields{
			"time": time.Now(),
			"uuid": logid,
		}).Error(logErr)
	}
	restaurantId := chi.URLParam(r, "restaurantId")
	var body models.AddDishesBody
	adminCtx := middlewares.UserContext(r)
	if parseErr := utils.ParseBody(r.Body, &body); parseErr != nil {
		utils.RespondError(w, http.StatusBadRequest, parseErr, "Failed to parse request body", body)
		return
	}

	exists, existsErr := dbHelper.IsRestaurantIDExists(restaurantId)
	if existsErr != nil {
		utils.RespondError(w, http.StatusInternalServerError, existsErr, "Failed to check Restaurant existence", body)
		return
	}

	if !exists {
		utils.RespondError(w, http.StatusBadRequest, nil, "Restaurant not exists", body)
		return
	}

	if body.Name == "" {
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Dish Name.", body)
		return
	}

	if body.Description == "" {
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Dish Description.", body)
		return
	}

	if body.Quantity <= 0 {
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Dish Quantity.", body)
		return
	}

	if body.Price <= 0 {
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Dish Price.", body)
		return
	}

	if body.Discount > 100 || body.Discount < 0 {
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Discount.", body)
		return
	}
	_, saveErr := dbHelper.CreateDish(restaurantId, adminCtx.ID, body.Name, body.Description, body.Quantity, body.Price, body.Discount)
	if saveErr != nil {
		utils.RespondError(w, http.StatusInternalServerError, saveErr, "Failed to add Restaurant Dish.", body)
		return
	}
	log.Logger.WithFields(log.Fields{
		"time":        time.Now(),
		"uuid":        logid.String(),
		"requestBody": body,
		"responseBody": models.Message{
			Message: "Restaurant Dish added successfully",
		},
	}).Info("Restaurant Dishes added successfully")
	utils.RespondJSON(w, http.StatusCreated, models.Message{
		Message: "Restaurant Dish added successfully",
	})
}

func UpdateDish(w http.ResponseWriter, r *http.Request) {
	logrus := log.GetLogger()
	logid, logErr := uuid.NewV4()
	if logErr != nil {
		log.Logger.WithFields(log.Fields{
			"time": time.Now(),
			"uuid": logid,
		}).Error(logErr)
	}
	restaurantId := chi.URLParam(r, "restaurantId")
	dishId := chi.URLParam(r, "dishId")
	var body models.AddDishesBody

	adminCtx := middlewares.UserContext(r)
	if parseErr := utils.ParseBody(r.Body, &body); parseErr != nil {
		utils.RespondError(w, http.StatusBadRequest, parseErr, "Failed to parse request body", body)
		return
	}

	dish, dishErr := dbHelper.GetDishByID(dishId)
	if dishErr != nil {
		utils.RespondError(w, http.StatusBadRequest, nil, "Dish not exist", body)
		return
	}

	if adminCtx.CurrentRole != models.RoleAdmin && dish.CreatedBy != adminCtx.ID {
		utils.RespondError(w, http.StatusBadRequest, nil, "Dish not Created by: "+string(adminCtx.CurrentRole), body)
		return
	}

	if body.Name == "" {
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Name.", body)
		return
	}

	if body.Description == "" {
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Description.", body)
		return
	}

	if body.Quantity <= 0 {
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Quantity.", body)
		return
	}

	if body.Price <= 0 {
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Price.", body)
		return
	}

	if body.Discount > 100 || body.Discount < 0 {
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Discount.", body)
		return
	}

	err := dbHelper.UpdateDish(dishId, restaurantId, body.Name, body.Description, body.Quantity, body.Price, body.Discount)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to update Restaurant Dish", body)
		return
	}
	log.Logger.WithFields(log.Fields{
		"time":        time.Now(),
		"uuid":        logid.String(),
		"requestBody": body,
		"responseBody": models.Message{
			Message: "Restaurant Dish Updated successfully",
		},
	}).Info("Restaurant Dish Updated successfully")
	utils.RespondJSON(w, http.StatusCreated, models.Message{
		Message: "Restaurant Dish Updated successfully",
	})
}

func RemoveDish(w http.ResponseWriter, r *http.Request) {
	logrus := log.GetLogger()
	logid, logErr := uuid.NewV4()
	if logErr != nil {
		log.Logger.WithFields(log.Fields{
			"time": time.Now(),
			"uuid": logid,
		}).Error(logErr)
	}
	restaurantId := chi.URLParam(r, "restaurantId")
	dishId := chi.URLParam(r, "dishId")
	adminCtx := middlewares.UserContext(r)
	if adminCtx.CurrentRole == models.RoleAdmin {
		err := dbHelper.RemoveDish(dishId, restaurantId)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get Restaurant Dish", "restaurantId: "+restaurantId+"dishId: "+dishId)
			return
		}
	} else {
		err := dbHelper.RemoveDishByUserID(dishId, restaurantId, adminCtx.ID)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get Restaurant Dish", "restaurantId: "+restaurantId+"dishId: "+dishId)
			return
		}
	}
	log.Logger.WithFields(log.Fields{
		"time":        time.Now(),
		"uuid":        logid.String(),
		"requestBody": "restaurantId: " + restaurantId + "dishId: " + dishId,
		"responseBody": models.Message{
			Message: "Restaurant Dish Removed successfully.",
		},
	}).Info("Restaurant Dish Removed successfully.")
	utils.RespondJSON(w, http.StatusCreated, models.Message{
		Message: "Restaurant Dish Removed successfully.",
	})
}

func GetRestaurantsDishes(w http.ResponseWriter, r *http.Request) {
	logrus := log.GetLogger()
	logid, logErr := uuid.NewV4()
	if logErr != nil {
		log.Logger.WithFields(log.Fields{
			"time": time.Now(),
			"uuid": logid,
		}).Error(logErr)
	}
	Filters := utils.GetDishFilters(r)
	restaurantId := chi.URLParam(r, "restaurantId")
	adminCtx := middlewares.UserContext(r)
	if adminCtx.CurrentRole == models.RoleAdmin || adminCtx.CurrentRole == models.RoleUser {
		DishesCount, DishesCountErr := dbHelper.GetRestaurantDishesCount(restaurantId, Filters)
		if DishesCountErr != nil {
			utils.RespondError(w, http.StatusInternalServerError, DishesCountErr, "Failed to get Restaurant Dishes Count.", Filters)
			return
		}
		Dishes, err := dbHelper.GetRestaurantDishes(restaurantId, Filters)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get Restaurant Dishes", Filters)
			return
		}
		log.Logger.WithFields(log.Fields{
			"time":        time.Now(),
			"uuid":        logid.String(),
			"requestBody": Filters,
			"responseBody": models.GetDishes{
				Message:    "Get Restaurants successfully.",
				Dishes:     Dishes,
				TotalCount: DishesCount,
				PageSize:   Filters.PageSize,
				PageNumber: Filters.PageNumber,
			},
		}).Info("Get Restaurants successfully.")
		utils.RespondJSON(w, http.StatusCreated, models.GetDishes{
			Message:    "Get Restaurants successfully.",
			Dishes:     Dishes,
			TotalCount: DishesCount,
			PageSize:   Filters.PageSize,
			PageNumber: Filters.PageNumber,
		})
	} else {
		DishesCount, DishesCountErr := dbHelper.GetRestaurantDishesCountByUserId(restaurantId, adminCtx.ID, Filters)
		if DishesCountErr != nil {
			utils.RespondError(w, http.StatusInternalServerError, DishesCountErr, "Failed to get Restaurant Dishes Count.", Filters)
			return
		}
		Dishes, err := dbHelper.GetRestaurantDishesByUserID(restaurantId, adminCtx.ID, Filters)
		if err != nil {
			utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get Restaurant Dish", Filters)
			return
		}
		log.Logger.WithFields(log.Fields{
			"time":        time.Now(),
			"uuid":        logid.String(),
			"requestBody": Filters,
			"responseBody": models.GetDishes{
				Message:    "Get Restaurants successfully.",
				Dishes:     Dishes,
				TotalCount: DishesCount,
				PageSize:   Filters.PageSize,
				PageNumber: Filters.PageNumber,
			},
		}).Info("Get Restaurants successfully.")
		utils.RespondJSON(w, http.StatusCreated, models.GetDishes{
			Message:    "Get Restaurants successfully.",
			Dishes:     Dishes,
			TotalCount: DishesCount,
			PageSize:   Filters.PageSize,
			PageNumber: Filters.PageNumber,
		})
	}
}
