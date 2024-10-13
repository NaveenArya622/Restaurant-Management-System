package handler

import (
	"net/http"
	"rms/database"
	"rms/database/dbHelper"
	"rms/middlewares"
	"rms/models"
	"rms/utils"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

func RegisterUser(w http.ResponseWriter, r *http.Request) {
	var body models.RegisterUserBody
	subAdminCtx := middlewares.UserContext(r)
	if parseErr := utils.ParseBody(r.Body, &body); parseErr != nil {
		logrus.Errorf("Failed to parse request body: %s", parseErr)
		utils.RespondError(w, http.StatusBadRequest, parseErr, "Failed to parse request body")
		return
	}
	if len(body.Password) < 6 {
		logrus.Errorf("password must be 6 chars long")
		utils.RespondError(w, http.StatusBadRequest, nil, "password must be 6 chars long")
		return
	}

	if !utils.IsEmailValid(body.Email) {
		logrus.Errorf("Invalid Email.")
		//TODO := you can generate custom error message by errors.new function if you want to generate an custom error
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Email.")
		return
	}

	exists, existsErr := dbHelper.IsUserRoleExists(body.Email, models.RoleUser)
	if existsErr != nil {
		logrus.Errorf("Failed to check user role existence")
		utils.RespondError(w, http.StatusInternalServerError, existsErr, "Failed to check user role existence")
		return
	}
	if exists {
		logrus.Errorf("user already exists")
		utils.RespondError(w, http.StatusBadRequest, nil, "user already exists")
		return
	}
	hashedPassword, hasErr := utils.HashPassword(body.Password)
	if hasErr != nil {
		logrus.Errorf("Failed to parse request body: %s", hasErr)
		utils.RespondError(w, http.StatusInternalServerError, hasErr, "Failed to secure password")
		return
	}
	//todo  use this db call outside transaction **DONE**
	userID, existsErr := dbHelper.IsUserExists(body.Email)
	if existsErr != nil {
		logrus.Errorf("Failed to parse request body: %s", existsErr)
		utils.RespondError(w, http.StatusInternalServerError, existsErr, "Failed to check user existence")
		return
	}
	txErr := database.Tx(func(tx *sqlx.Tx) error {
		if len(userID) > 0 {
			roleErr := dbHelper.CreateUserRole(tx, userID, subAdminCtx.ID, models.RoleUser)
			if roleErr != nil {
				logrus.Errorf("Failed to parse request body: %s", roleErr)
				return roleErr
			}
		} else {
			userID, saveErr := dbHelper.CreateUser(tx, body.Name, body.Email, hashedPassword)
			if saveErr != nil {
				logrus.Errorf("Failed to parse request body: %s", saveErr)
				return saveErr
			}
			roleErr := dbHelper.CreateUserRole(tx, userID, subAdminCtx.ID, models.RoleUser)
			if roleErr != nil {
				logrus.Errorf("Failed to parse request body: %s", roleErr)
				return roleErr
			}
		}
		return nil
	})
	if txErr != nil {
		logrus.Errorf("Failed to create user: %s", txErr)
		utils.RespondError(w, http.StatusInternalServerError, txErr, "Failed to create user")
		return
	}
	logrus.Infof("user Created successfully.")
	utils.RespondJSON(w, http.StatusCreated, models.Message{
		Message: "user Created successfully",
	})
}

func GetUsers(w http.ResponseWriter, r *http.Request) {
	//ToDO change the name of filters to samll letter
	Filters := utils.GetFilters(r)
	if Filters.Email != "" && !utils.IsEmailValid(Filters.Email) {
		logrus.Errorf("Invalid Filter Email.")
		//TODO Why this statuscode is return
		utils.RespondError(w, http.StatusExpectationFailed, nil, "Invalid Filter Email.")
		return
	}
	adminCtx := middlewares.UserContext(r)
	var UserCount int64
	var Users []models.User
	var errGroup errgroup.Group
	if adminCtx.CurrentRole == models.RoleAdmin {
		errGroup.Go(func() error {
			var err error
			UserCount, err = dbHelper.GetUserCount(models.RoleUser, Filters)
			logrus.Errorf("Unable to get Users Count: %s", err)
			return err
		})
		errGroup.Go(func() error {
			var err error
			Users, err = dbHelper.GetUsers(models.RoleUser, Filters)
			logrus.Errorf("Unable to get Users: %s", err)
			return err
		})
	} else {
		errGroup.Go(func() error {
			var err error
			UserCount, err = dbHelper.GetUserCountByAdminID(adminCtx.ID, models.RoleUser, Filters)
			logrus.Errorf("Unable to get Users Count: %s", err)
			return err
		})
		errGroup.Go(func() error {
			var err error
			Users, err = dbHelper.GetUsersByAdminID(adminCtx.ID, models.RoleUser, Filters)
			logrus.Errorf("Unable to get Users: %s", err)
			return err
		})
	}
	if err := errGroup.Wait(); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Unable to get Users")
		return
	}
	logrus.Infof("Get users successfully.")
	utils.RespondJSON(w, http.StatusOK, models.GetUsers{
		Message:    "Get users successfully.",
		Users:      Users,
		TotalCount: UserCount,
		PageSize:   Filters.PageSize,
		PageNumber: Filters.PageNumber,
	})
}

// todo :- remove user details also **DONE**
func RemoveUser(w http.ResponseWriter, r *http.Request) {
	//todo :- remove user details **DONE**
	id := chi.URLParam(r, "userId")
	adminCtx := middlewares.UserContext(r)
	multipleRoles, rolesErr := dbHelper.UserHaveMultipleRoles(id)
	if rolesErr != nil {
		logrus.Errorf("Unable to get Users: %s", rolesErr)
		utils.RespondError(w, http.StatusInternalServerError, rolesErr, "Unable to get Users")
		return
	}
	//TODO this can be written into single transaction for removing redundancy
	if adminCtx.CurrentRole == models.RoleAdmin {
		txErr := database.Tx(func(tx *sqlx.Tx) error {
			if multipleRoles {
				roleErr := dbHelper.RemoveRole(id, models.RoleUser)
				if roleErr != nil {
					logrus.Errorf("Failed to Remove User Role: %s", roleErr)
					return roleErr
				}
			} else {
				roleErr := dbHelper.RemoveRole(id, models.RoleUser)
				if roleErr != nil {
					logrus.Errorf("Failed to Remove User Role: %s", roleErr)
					return roleErr
				}
				userErr := dbHelper.RemoveUser(id)
				if userErr != nil {
					logrus.Errorf("Failed to remove User: %s", userErr)
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
	} else {
		txErr := database.Tx(func(tx *sqlx.Tx) error {
			// TODO use tx inside the transaction
			if multipleRoles {
				saveErr := dbHelper.RemoveRoleByAdminID(id, adminCtx.ID, models.RoleUser)
				if saveErr != nil {
					logrus.Errorf("Failed to parse request body: %s", saveErr)
					return saveErr
				}
			} else {
				saveErr := dbHelper.RemoveRoleByAdminID(id, adminCtx.ID, models.RoleUser)
				if saveErr != nil {
					logrus.Errorf("Failed to parse request body: %s", saveErr)
					return saveErr
				}
				roleErr := dbHelper.RemoveUser(id)
				if roleErr != nil {
					logrus.Errorf("Failed to parse request body: %s", roleErr)
					return roleErr
				}
			}
			return nil
		})
		if txErr != nil {
			logrus.Errorf("Failed to create user: %s", txErr)
			utils.RespondError(w, http.StatusInternalServerError, txErr, "Failed to create user")
			return
		}
	}
	logrus.Infof("User removed successfully.")
	utils.RespondJSON(w, http.StatusOK, struct {
		Message string `json:"message"`
	}{
		Message: "User removed successfully.",
	})
}

// Restaurant

func OpenRestaurant(w http.ResponseWriter, r *http.Request) {
	var body models.OpenRestaurantBody
	adminCtx := middlewares.UserContext(r)
	if parseErr := utils.ParseBody(r.Body, &body); parseErr != nil {
		logrus.Errorf("Failed to parse request body: %s", parseErr)
		utils.RespondError(w, http.StatusBadRequest, parseErr, "Failed to parse request body")
		return
	}

	if !utils.IsEmailValid(body.Email) {
		logrus.Errorf("Invalid Email.")
		utils.RespondError(w, http.StatusExpectationFailed, nil, "Invalid Email.")
		return
	}

	exists, existsErr := dbHelper.IsRestaurantExists(body.Email)
	if existsErr != nil {
		logrus.Errorf("Failed to check Restaurant existence: %s", existsErr)
		utils.RespondError(w, http.StatusInternalServerError, existsErr, "Failed to check Restaurant existence")
		return
	}

	if exists {
		logrus.Errorf("Restaurant already exists.")
		utils.RespondError(w, http.StatusAlreadyReported, nil, "Restaurant already exists")
		return
	}

	if body.Name == "" {
		logrus.Errorf("Invalid Name.")
		utils.RespondError(w, http.StatusExpectationFailed, nil, "Invalid Name")
		return
	}
	if !utils.IsEmailValid(body.Email) {
		logrus.Errorf("Invalid Email.")
		utils.RespondError(w, http.StatusExpectationFailed, nil, "Invalid Email")
		return
	}

	if len(body.Address) > 30 || len(body.Address) <= 2 {
		logrus.Errorf("Address must be with in 2 to 30 letter.")
		utils.RespondError(w, http.StatusExpectationFailed, nil, "Address must be with in 2 to 30 letter.")
		return
	}

	if len(body.State) > 16 || len(body.State) <= 2 {
		logrus.Errorf("State must be with in 2 to 16 letter.")
		utils.RespondError(w, http.StatusExpectationFailed, nil, "State must be with in 2 to 16 letter.")
		return
	}

	if len(body.City) > 20 || len(body.City) <= 2 {
		logrus.Errorf("City must be with in 2 to 20 letter.")
		utils.RespondError(w, http.StatusExpectationFailed, nil, "City must be with in 2 to 20 letter.")
		return
	}

	if len(body.PinCode) != 6 {
		logrus.Errorf("PinCode must 6 digit.")
		utils.RespondError(w, http.StatusExpectationFailed, nil, "PinCode must 6 digit.")
		return
	}

	if body.Lat > 90 || body.Lat < -90 {
		logrus.Errorf("Invalid Latitude.")
		utils.RespondError(w, http.StatusExpectationFailed, nil, "Invalid Latitude.")
		return
	}

	if body.Lng > 180 || body.Lng < -180 {
		logrus.Errorf("Invalid Longitude.")
		utils.RespondError(w, http.StatusExpectationFailed, nil, "Invalid Longitude.")
		return
	}
	_, saveErr := dbHelper.CreateRestaurant(body.Name, body.Email, adminCtx.ID, body.Address, body.State, body.City, body.PinCode, body.Lat, body.Lng)
	if saveErr != nil {
		logrus.Errorf("Failed to open Restaurant: %s", saveErr)
		utils.RespondError(w, http.StatusInternalServerError, saveErr, "Failed to open Restaurant")
		return
	}
	logrus.Infof("Restaurant Opened successfully")
	utils.RespondJSON(w, http.StatusOK, models.Message{
		Message: "Restaurant Opened successfully",
	})
}

func CloseRestaurant(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "restaurantId")
	adminCtx := middlewares.UserContext(r)
	var err error
	if adminCtx.CurrentRole == models.RoleAdmin {
		err = dbHelper.CloseRestaurant(id)
	} else {
		err = dbHelper.CloseMyRestaurant(id, adminCtx.ID)
	}
	if err != nil {
		logrus.Errorf("Unable to get Restaurant: %s", err)
		utils.RespondError(w, http.StatusInternalServerError, err, "Unable to get Restaurant")
		return
	}
	logrus.Infof("Restaurant Closed successfully.")
	utils.RespondJSON(w, http.StatusOK, models.Message{
		Message: "Restaurant Closed successfully.",
	})
}

func GetRestaurants(w http.ResponseWriter, r *http.Request) {
	Filters := utils.GetFilters(r)
	if Filters.Email != "" && !utils.IsEmailValid(Filters.Email) {
		logrus.Errorf("Invalid Restaurants Filter Email.")
		utils.RespondError(w, http.StatusInternalServerError, nil, "Invalid Restaurants Filter Email.")
		return
	}
	adminCtx := middlewares.UserContext(r)

	//TODO use errgroup when mutiple db calls for less time consuming **DONE**

	//TODO follow camelCase convention in variable naming until it has public access for ex RestaurantsCount is restaurantsCount(like this)
	var RestaurantsCount int64
	var Restaurants []models.Restaurant
	var errGroup errgroup.Group
	if adminCtx.CurrentRole == models.RoleAdmin || adminCtx.CurrentRole == models.RoleUser {
		errGroup.Go(func() error {
			var err error
			RestaurantsCount, err = dbHelper.GetRestaurantsCount(Filters)
			logrus.Errorf("Unable to get Restaurants Count: %s", err)
			return err
		})
		errGroup.Go(func() error {
			var err error
			Restaurants, err = dbHelper.GetRestaurants(Filters)
			logrus.Errorf("Unable to get Restaurants: %s", err)
			return err
		})
	} else {
		errGroup.Go(func() error {
			var err error
			RestaurantsCount, err = dbHelper.GetRestaurantsCountByUserID(adminCtx.ID, Filters)
			logrus.Errorf("Unable to get Restaurants Count: %s", err)
			return err
		})
		errGroup.Go(func() error {
			var err error
			Restaurants, err = dbHelper.GetRestaurantsByUserID(adminCtx.ID, Filters)
			logrus.Errorf("Unable to get Restaurants: %s", err)
			return err
		})
	}
	if err := errGroup.Wait(); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, err, "Unable to get Restaurants")
		return
	}
	//TODO  write this  message below else one time **DONE**
	logrus.Infof("Get Restaurants successfully.")
	utils.RespondJSON(w, http.StatusCreated, models.GetRestaurants{
		Message:     "Get Restaurants successfully.",
		Restaurants: Restaurants,
		TotalCount:  RestaurantsCount,
		PageNumber:  Filters.PageNumber,
		PageSize:    Filters.PageSize,
	})
}

func UpdateRestaurant(w http.ResponseWriter, r *http.Request) {
	restaurantId := chi.URLParam(r, "restaurantId")
	var body models.OpenRestaurantBody

	adminCtx := middlewares.UserContext(r)
	if parseErr := utils.ParseBody(r.Body, &body); parseErr != nil {
		logrus.Errorf("Failed to parse request body: %s", parseErr)
		utils.RespondError(w, http.StatusBadRequest, parseErr, "Failed to parse request body")
		return
	}

	restaurant, restaurantErr := dbHelper.GetRestaurantByID(restaurantId)
	if restaurantErr != nil {
		logrus.Errorf("Restaurant not exist: %s", restaurantErr)
		utils.RespondError(w, http.StatusBadRequest, restaurantErr, "Restaurant not exist")
		return
	}

	if adminCtx.CurrentRole != models.RoleAdmin && restaurant.CreatedBy != adminCtx.ID {
		logrus.Errorf("Restaurant not Created by: %s", adminCtx.CurrentRole)
		utils.RespondError(w, http.StatusBadRequest, nil, "Restaurant not Created by: "+string(adminCtx.CurrentRole))
		return
	}

	if body.Name == "" {
		body.Name = restaurant.Name
	}
	if !utils.IsEmailValid(body.Email) {
		if body.Email != "" {
			logrus.Errorf("Invalid Email.")
			utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Email")
			return
		}
		body.Email = restaurant.Email
	}

	if len(body.Address) > 30 || len(body.Address) <= 2 {
		if body.Address != "" {
			logrus.Errorf("Address must be with in 2 to 30 letter.")
			utils.RespondError(w, http.StatusBadRequest, nil, "Address must be with in 2 to 30 letter.")
			return
		}
		body.Address = restaurant.Address
	}

	if len(body.State) > 16 || len(body.State) <= 2 {
		if body.State != "" {
			logrus.Errorf("State must be with in 2 to 16 letter.")
			utils.RespondError(w, http.StatusBadRequest, nil, "State must be with in 2 to 16 letter.")
			return
		}
		body.State = restaurant.State
	}

	if len(body.City) > 20 || len(body.City) <= 2 {
		if body.City != "" {
			logrus.Errorf("City must be with in 2 to 20 letter.")
			utils.RespondError(w, http.StatusBadRequest, nil, "City must be with in 2 to 20 letter.")
			return
		}
		body.City = restaurant.City
	}

	if len(body.PinCode) != 6 {
		if body.PinCode != "" {
			logrus.Errorf("PinCode must 6 digit.")
			utils.RespondError(w, http.StatusBadRequest, nil, "PinCode must 6 digit.")
			return
		}
		body.PinCode = restaurant.PinCode
	}

	if body.Lat > 90 || body.Lat < -90 {
		if body.Lat != 0 {
			logrus.Errorf("Invalid Latitude.")
			utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Latitude.")
			return
		}
		body.Lat = restaurant.Lat
	}

	if body.Lng > 180 || body.Lng < -180 {
		if body.Lat != 0 {
			logrus.Errorf("Invalid Longitude.")
			utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Longitude.")
			return
		}
		body.Lng = restaurant.Lng
	}

	err := dbHelper.UpdateRestaurant(restaurantId, body.Name, body.Email, body.Address, body.State, body.City, body.PinCode, body.Lat, body.Lng)
	if err != nil {
		logrus.Errorf("Failed to update Restaurant: %s", err)
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to update Restaurant")
		return
	}
	logrus.Infof("Restaurant Opened successfully.")
	utils.RespondJSON(w, http.StatusCreated, models.Message{
		Message: "Restaurant Updated successfully.",
	})
}

// Restaurant Dishes

func AddRestaurantDish(w http.ResponseWriter, r *http.Request) {
	restaurantId := chi.URLParam(r, "restaurantId")
	var body models.AddDishesBody
	adminCtx := middlewares.UserContext(r)
	if parseErr := utils.ParseBody(r.Body, &body); parseErr != nil {
		logrus.Errorf("Failed to parse request body: %s", parseErr)
		utils.RespondError(w, http.StatusBadRequest, parseErr, "Failed to parse request body")
		return
	}

	exists, existsErr := dbHelper.IsRestaurantIDExists(restaurantId)
	if existsErr != nil {
		logrus.Errorf("Failed to parse request body: %s", existsErr)
		utils.RespondError(w, http.StatusInternalServerError, existsErr, "Failed to check Restaurant existence")
		return
	}

	if !exists {
		logrus.Errorf("Restaurant not exists.")
		utils.RespondError(w, http.StatusBadRequest, nil, "Restaurant not exists")
		return
	}

	if body.Name == "" {
		logrus.Errorf("Invalid Dish Name.")
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Dish Name.")
		return
	}

	if body.Description == "" {
		logrus.Errorf("Invalid Dish Description.")
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Dish Description.")
		return
	}

	if body.Quantity <= 0 {
		logrus.Errorf("Invalid Dish Quantity.")
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Dish Quantity.")
		return
	}

	if body.Price <= 0 {
		logrus.Errorf("Invalid Dish Price.")
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Dish Price.")
		return
	}

	if body.Discount > 100 || body.Discount < 0 {
		logrus.Errorf("Invalid Discount.")
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Discount.")
		return
	}
	_, saveErr := dbHelper.CreateDish(restaurantId, adminCtx.ID, body.Name, body.Description, body.Quantity, body.Price, body.Discount)
	if saveErr != nil {
		logrus.Errorf("Failed to add Restaurant Dish: %s", saveErr)
		utils.RespondError(w, http.StatusInternalServerError, saveErr, "Failed to add Restaurant Dish.")
		return
	}
	logrus.Infof("Restaurant Dishes added successfully")
	utils.RespondJSON(w, http.StatusCreated, models.Message{
		Message: "Restaurant Dish added successfully",
	})
}

func UpdateDish(w http.ResponseWriter, r *http.Request) {
	restaurantId := chi.URLParam(r, "restaurantId")
	dishId := chi.URLParam(r, "dishId")
	var body models.AddDishesBody

	adminCtx := middlewares.UserContext(r)
	if parseErr := utils.ParseBody(r.Body, &body); parseErr != nil {
		logrus.Errorf("Failed to parse request body: %s", parseErr)
		utils.RespondError(w, http.StatusBadRequest, parseErr, "Failed to parse request body")
		return
	}

	dish, dishErr := dbHelper.GetDishByID(dishId)
	if dishErr != nil {
		logrus.Errorf("Dish not exist: %s", dishErr)
		utils.RespondError(w, http.StatusBadRequest, nil, "Dish not exist")
		return
	}

	if adminCtx.CurrentRole != models.RoleAdmin && dish.CreatedBy != adminCtx.ID {
		logrus.Errorf("Dish not Created by %s", adminCtx.CurrentRole)
		utils.RespondError(w, http.StatusBadRequest, nil, "Dish not Created by: "+string(adminCtx.CurrentRole))
		return
	}

	if body.Name == "" {
		body.Name = dish.Name
	}

	if body.Description == "" {
		body.Name = dish.Description
	}

	if body.Quantity <= 0 {
		body.Quantity = dish.Quantity
	}

	if body.Price <= 0 {
		body.Price = dish.Price
	}

	if body.Discount > 100 || body.Discount < 0 {
		logrus.Errorf("Invalid Discount.")
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Discount.")
		return
	}

	err := dbHelper.UpdateDish(dishId, restaurantId, body.Name, body.Description, body.Quantity, body.Price, body.Discount)
	if err != nil {
		logrus.Errorf("Failed update Restaurant Dish: %s", err)
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to update Restaurant Dish")
		return
	}
	logrus.Infof("Restaurant Dish Updated successfully")
	utils.RespondJSON(w, http.StatusCreated, models.Message{
		Message: "Restaurant Dish Updated successfully",
	})
}

func RemoveDish(w http.ResponseWriter, r *http.Request) {
	restaurantId := chi.URLParam(r, "restaurantId")
	dishId := chi.URLParam(r, "dishId")
	adminCtx := middlewares.UserContext(r)
	if adminCtx.CurrentRole == models.RoleAdmin {
		err := dbHelper.RemoveDish(dishId, restaurantId)
		if err != nil {
			logrus.Errorf("Failed to get Restaurant Dish: %s", err)
			utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get Restaurant Dish")
			return
		}
	} else {
		err := dbHelper.RemoveDishByUserID(dishId, restaurantId, adminCtx.ID)
		if err != nil {
			logrus.Errorf("Failed to get Restaurant Dish: %s", err)
			utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get Restaurant Dish")
			return
		}
	}
	logrus.Infof("Restaurant Dish Removed successfully.")
	utils.RespondJSON(w, http.StatusCreated, models.Message{
		Message: "Restaurant Dish Removed successfully.",
	})
}

func GetRestaurantsDishes(w http.ResponseWriter, r *http.Request) {
	Filters := utils.GetDishFilters(r)
	restaurantId := chi.URLParam(r, "restaurantId")
	adminCtx := middlewares.UserContext(r)
	if adminCtx.CurrentRole == models.RoleAdmin || adminCtx.CurrentRole == models.RoleUser {
		DishesCount, DishesCountErr := dbHelper.GetRestaurantDishesCount(restaurantId, Filters)
		if DishesCountErr != nil {
			logrus.Errorf("Failed to get Restaurant Dishes Count: %s", DishesCountErr)
			utils.RespondError(w, http.StatusInternalServerError, DishesCountErr, "Failed to get Restaurant Dishes Count.")
			return
		}
		Dishes, err := dbHelper.GetRestaurantDishes(restaurantId, Filters)
		if err != nil {
			logrus.Errorf("Failed to get Restaurant Dishes: %s", err)
			utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get Restaurant Dishes")
			return
		}
		logrus.Errorf("Get Restaurants successfully.")
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
			logrus.Errorf("Failed to get Restaurant Dishes Count: %s", DishesCountErr)
			utils.RespondError(w, http.StatusInternalServerError, DishesCountErr, "Failed to get Restaurant Dishes Count.")
			return
		}
		Dishes, err := dbHelper.GetRestaurantDishesByUserID(restaurantId, adminCtx.ID, Filters)
		if err != nil {
			logrus.Errorf("Failed to get Restaurant Dishes: %s", err)
			utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get Restaurant Dish")
			return
		}
		logrus.Errorf("Get Restaurants successfully.")
		utils.RespondJSON(w, http.StatusCreated, models.GetDishes{
			Message:    "Get Restaurants successfully.",
			Dishes:     Dishes,
			TotalCount: DishesCount,
			PageSize:   Filters.PageSize,
			PageNumber: Filters.PageNumber,
		})
	}
}
