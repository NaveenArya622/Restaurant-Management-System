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
)

func RegisterUser(w http.ResponseWriter, r *http.Request) {
	var body models.RegisterUserBody
	subAdminCtx := middlewares.UserContext(r)
	if parseErr := utils.ParseBody(r.Body, &body); parseErr != nil {
		logrus.Printf("Failed to parse request body: %s", parseErr)
		utils.RespondError(w, http.StatusBadRequest, parseErr, "Failed to parse request body")
		return
	}
	if len(body.Password) < 6 {
		logrus.Printf("password must be 6 chars long")
		utils.RespondError(w, http.StatusBadRequest, nil, "password must be 6 chars long")
		return
	}

	if !utils.IsEmailValid(body.Email) {
		logrus.Printf("Invalid Email.")
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Email.")
		return
	}

	exists, existsErr := dbHelper.IsUserRoleExists(body.Email, models.RoleUser)
	if existsErr != nil {
		logrus.Printf("Failed to check user role existence")
		utils.RespondError(w, http.StatusInternalServerError, existsErr, "Failed to check user role existence")
		return
	}
	if exists {
		logrus.Printf("user already exists")
		utils.RespondError(w, http.StatusBadRequest, nil, "user already exists")
		return
	}
	hashedPassword, hasErr := utils.HashPassword(body.Password)
	if hasErr != nil {
		logrus.Printf("Failed to parse request body: %s", hasErr)
		utils.RespondError(w, http.StatusInternalServerError, hasErr, "Failed to secure password")
		return
	}

	txErr := database.Tx(func(tx *sqlx.Tx) error {

		// IsUserExists is a independent query so dont put it in transaction

		userID, existsErr := dbHelper.IsUserExists(body.Email)
		if existsErr != nil {
			logrus.Printf("Failed to parse request body: %s", existsErr)

			// no need to send respond here

			utils.RespondError(w, http.StatusInternalServerError, existsErr, "Failed to check user existence")
			return existsErr
		}

		// why are you calling createuserRole two times

		if len(userID) > 0 {
			roleErr := dbHelper.CreateUserRole(tx, userID, subAdminCtx.ID, models.RoleUser)
			if roleErr != nil {
				logrus.Printf("Failed to parse request body: %s", roleErr)
				return roleErr
			}
		} else {
			userID, saveErr := dbHelper.CreateUser(tx, body.Name, body.Email, hashedPassword)
			if saveErr != nil {
				logrus.Printf("Failed to parse request body: %s", saveErr)
				return saveErr
			}
			roleErr := dbHelper.CreateUserRole(tx, userID, subAdminCtx.ID, models.RoleUser)
			if roleErr != nil {
				logrus.Printf("Failed to parse request body: %s", roleErr)
				return roleErr
			}
		}
		return nil
	})
	if txErr != nil {
		logrus.Printf("Failed to create user: %s", txErr)
		utils.RespondError(w, http.StatusInternalServerError, txErr, "Failed to create user")
		return
	}
	logrus.Printf("user Created successfully.")
	utils.RespondJSON(w, http.StatusCreated, models.Message{
		Message: "user Created successfully",
	})
}

func GetUsers(w http.ResponseWriter, r *http.Request) {
	Filters := utils.GetFilters(r)
	if Filters.Email != "" && !utils.IsEmailValid(Filters.Email) {
		logrus.Printf("Invalid Filter Email.")
		utils.RespondError(w, http.StatusInternalServerError, nil, "Invalid Filter Email.")
		return
	}

	/*
		there is no need of if else and writing this much big code
		get the logged used id and simply return the users accordingly.
	*/

	adminCtx := middlewares.UserContext(r)
	if adminCtx.CurrentRole == models.RoleAdmin {
		UserCount, countErr := dbHelper.GetUserCount(models.RoleUser, Filters)
		if countErr != nil {
			logrus.Printf("Unable to get Users: %s", countErr)
			utils.RespondError(w, http.StatusInternalServerError, countErr, "Unable to get Users")
			return
		}
		Users, err := dbHelper.GetUsers(models.RoleUser, Filters)
		if err != nil {
			logrus.Printf("Unable to get Users: %s", err)
			utils.RespondError(w, http.StatusInternalServerError, err, "Unable to get Users")
			return
		}

		// status code should be ok

		logrus.Printf("Get users successfully.")
		utils.RespondJSON(w, http.StatusCreated, models.GetUsers{
			Message:    "Get users successfully.",
			Users:      Users,
			TotalCount: UserCount,
			PageSize:   Filters.PageSize,
			PageNumber: Filters.PageNumber,
		})
	} else {
		UserCount, countErr := dbHelper.GetUserCountByAdminID(adminCtx.ID, models.RoleUser, Filters)
		if countErr != nil {
			logrus.Printf("Unable to get Users: %s", countErr)
			utils.RespondError(w, http.StatusInternalServerError, countErr, "Unable to get Users")
			return
		}
		Users, err := dbHelper.GetUsersByAdminID(adminCtx.ID, models.RoleUser, Filters)
		if err != nil {
			logrus.Printf("Unable to get Users: %s", err)
			utils.RespondError(w, http.StatusInternalServerError, err, "Unable to get Users")
			return
		}
		logrus.Printf("Get users successfully.")
		utils.RespondJSON(w, http.StatusCreated, models.GetUsers{
			Message:    "Get users successfully.",
			Users:      Users,
			TotalCount: UserCount,
			PageSize:   Filters.PageSize,
			PageNumber: Filters.PageNumber,
		})
	}
}

func RemoveUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "userId")
	adminCtx := middlewares.UserContext(r)

	/*
		same here also no need to put if else
	*/

	if adminCtx.CurrentRole == models.RoleAdmin {
		err := dbHelper.RemoveUser(id, models.RoleUser)
		if err != nil {
			logrus.Printf("Unable to get Users: %s", err)
			utils.RespondError(w, http.StatusInternalServerError, err, "Unable to get Users")
			return
		}
		logrus.Printf("User removed successfully.")
		utils.RespondJSON(w, http.StatusCreated, struct {
			Message string `json:"message"`
		}{
			Message: "User removed successfully.",
		})
	} else {
		err := dbHelper.RemoveUserByAdminID(id, adminCtx.ID, models.RoleUser)
		if err != nil {
			logrus.Printf("Unable to get Users: %s", err)
			utils.RespondError(w, http.StatusInternalServerError, err, "Unable to get Users")
			return
		}
		logrus.Printf("User removed successfully.")
		utils.RespondJSON(w, http.StatusCreated, models.Message{
			Message: "User removed successfully.",
		})
	}
}

// Restaurant

func OpenRestaurant(w http.ResponseWriter, r *http.Request) {
	var body models.OpenRestaurantBody
	adminCtx := middlewares.UserContext(r)
	if parseErr := utils.ParseBody(r.Body, &body); parseErr != nil {
		logrus.Printf("Failed to parse request body: %s", parseErr)
		utils.RespondError(w, http.StatusBadRequest, parseErr, "Failed to parse request body")
		return
	}

	if !utils.IsEmailValid(body.Email) {
		logrus.Printf("Invalid Email.")
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Email.")
		return
	}

	exists, existsErr := dbHelper.IsRestaurantExists(body.Email)
	if existsErr != nil {
		logrus.Printf("Failed to check Restaurant existence: %s", existsErr)
		utils.RespondError(w, http.StatusInternalServerError, existsErr, "Failed to check Restaurant existence")
		return
	}

	if exists {
		logrus.Printf("Restaurant already exists.")
		utils.RespondError(w, http.StatusBadRequest, nil, "Restaurant already exists")
		return
	}

	if body.Name == "" {
		logrus.Printf("Invalid Name.")
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Name")
		return
	}
	if !utils.IsEmailValid(body.Email) {
		logrus.Printf("Invalid Email.")
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Email")
		return
	}

	if len(body.Address) > 30 || len(body.Address) <= 2 {
		logrus.Printf("Address must be with in 2 to 30 letter.")
		utils.RespondError(w, http.StatusBadRequest, nil, "Address must be with in 2 to 30 letter.")
		return
	}

	if len(body.State) > 16 || len(body.State) <= 2 {
		logrus.Printf("State must be with in 2 to 16 letter.")
		utils.RespondError(w, http.StatusBadRequest, nil, "State must be with in 2 to 16 letter.")
		return
	}

	if len(body.City) > 20 || len(body.City) <= 2 {
		logrus.Printf("City must be with in 2 to 20 letter.")
		utils.RespondError(w, http.StatusBadRequest, nil, "City must be with in 2 to 20 letter.")
		return
	}

	if len(body.PinCode) != 6 {
		logrus.Printf("PinCode must 6 digit.")
		utils.RespondError(w, http.StatusBadRequest, nil, "PinCode must 6 digit.")
		return
	}

	if body.Lat > 90 || body.Lat < -90 {
		logrus.Printf("Invalid Latitude.")
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Latitude.")
		return
	}

	if body.Lng > 180 || body.Lng < -180 {
		logrus.Printf("Invalid Longitude.")
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Longitude.")
		return
	}
	_, saveErr := dbHelper.CreateRestaurant(body.Name, body.Email, adminCtx.ID, body.Address, body.State, body.City, body.PinCode, body.Lat, body.Lng)
	if saveErr != nil {
		logrus.Printf("Failed to open Restaurant: %s", saveErr)
		utils.RespondError(w, http.StatusBadRequest, saveErr, "Failed to open Restaurant")
		return
	}
	logrus.Printf("Restaurant Opened successfully")
	utils.RespondJSON(w, http.StatusCreated, models.Message{
		Message: "Restaurant Opened successfully",
	})
}

func CloseRestaurant(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "restaurantId")
	adminCtx := middlewares.UserContext(r)
	if adminCtx.CurrentRole == models.RoleAdmin {
		err := dbHelper.CloseRestaurant(id)
		if err != nil {
			logrus.Printf("Unable to get Restaurant: %s", err)
			utils.RespondError(w, http.StatusInternalServerError, err, "Unable to get Restaurant")
			return
		}
	} else {
		err := dbHelper.CloseMyRestaurant(id, adminCtx.ID)
		if err != nil {
			logrus.Printf("Unable to get Restaurant: %s", err)
			utils.RespondError(w, http.StatusInternalServerError, err, "Unable to get Restaurant")
			return
		}
	}
	logrus.Printf("Restaurant Closed successfully.")
	utils.RespondJSON(w, http.StatusCreated, models.Message{
		Message: "Restaurant Closed successfully.",
	})
}

func GetRestaurants(w http.ResponseWriter, r *http.Request) {
	Filters := utils.GetFilters(r)
	if Filters.Email != "" && !utils.IsEmailValid(Filters.Email) {
		logrus.Printf("Invalid Restaurants Filter Email.")
		utils.RespondError(w, http.StatusInternalServerError, nil, "Invalid Restaurants Filter Email.")
		return
	}
	adminCtx := middlewares.UserContext(r)
	if adminCtx.CurrentRole == models.RoleAdmin || adminCtx.CurrentRole == models.RoleUser {
		RestaurantsCount, countErr := dbHelper.GetRestaurantsCount(Filters)
		if countErr != nil {
			logrus.Printf("Unable to get Restaurants Count: %s", countErr)
			utils.RespondError(w, http.StatusInternalServerError, countErr, "Unable to get Restaurants Count.")
			return
		}
		Restaurants, err := dbHelper.GetRestaurants(Filters)
		if err != nil {
			logrus.Printf("Unable to get Restaurants: %s", err)
			utils.RespondError(w, http.StatusInternalServerError, err, "Unable to get Restaurants")
			return
		}
		logrus.Printf("Get Restaurants successfully.")
		utils.RespondJSON(w, http.StatusCreated, models.GetRestaurants{
			Message:     "Get Restaurants successfully.",
			Restaurants: Restaurants,
			TotalCount:  RestaurantsCount,
			PageNumber:  Filters.PageNumber,
			PageSize:    Filters.PageSize,
		})
	} else {
		RestaurantsCount, countErr := dbHelper.GetRestaurantsCountByUserID(adminCtx.ID, Filters)
		if countErr != nil {
			logrus.Printf("Unable to get Restaurants Count: %s", countErr)
			utils.RespondError(w, http.StatusInternalServerError, countErr, "Unable to get Restaurants Count.")
			return
		}
		Restaurants, err := dbHelper.GetRestaurantsByUserID(adminCtx.ID, Filters)
		if err != nil {
			logrus.Printf("Unable to get Restaurants: %s", err)
			utils.RespondError(w, http.StatusInternalServerError, err, "Unable to get Restaurant")
			return
		}
		logrus.Printf("Get Restaurants successfully.")
		utils.RespondJSON(w, http.StatusCreated, models.GetRestaurants{
			Message:     "Get Restaurants successfully.",
			Restaurants: Restaurants,
			TotalCount:  RestaurantsCount,
			PageNumber:  Filters.PageNumber,
			PageSize:    Filters.PageSize,
		})
	}
}

func UpdateRestaurant(w http.ResponseWriter, r *http.Request) {
	restaurantId := chi.URLParam(r, "restaurantId")
	var body models.OpenRestaurantBody

	adminCtx := middlewares.UserContext(r)
	if parseErr := utils.ParseBody(r.Body, &body); parseErr != nil {
		logrus.Printf("Failed to parse request body: %s", parseErr)
		utils.RespondError(w, http.StatusBadRequest, parseErr, "Failed to parse request body")
		return
	}

	restaurant, restaurantErr := dbHelper.GetRestaurantByID(restaurantId)
	if restaurantErr != nil {
		logrus.Printf("Restaurant not exist: %s", restaurantErr)
		utils.RespondError(w, http.StatusBadRequest, restaurantErr, "Restaurant not exist")
		return
	}

	if adminCtx.CurrentRole != models.RoleAdmin && restaurant.CreatedBy != adminCtx.ID {
		logrus.Printf("Restaurant not Created by: %s", adminCtx.CurrentRole)
		utils.RespondError(w, http.StatusBadRequest, nil, "Restaurant not Created by: "+string(adminCtx.CurrentRole))
		return
	}

	if body.Name == "" {
		body.Name = restaurant.Name
	}
	if !utils.IsEmailValid(body.Email) {
		if body.Email != "" {
			logrus.Printf("Invalid Email.")
			utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Email")
			return
		}
		body.Email = restaurant.Email
	}

	if len(body.Address) > 30 || len(body.Address) <= 2 {
		if body.Address != "" {
			logrus.Printf("Address must be with in 2 to 30 letter.")
			utils.RespondError(w, http.StatusBadRequest, nil, "Address must be with in 2 to 30 letter.")
			return
		}
		body.Address = restaurant.Address
	}

	if len(body.State) > 16 || len(body.State) <= 2 {
		if body.State != "" {
			logrus.Printf("State must be with in 2 to 16 letter.")
			utils.RespondError(w, http.StatusBadRequest, nil, "State must be with in 2 to 16 letter.")
			return
		}
		body.State = restaurant.State
	}

	if len(body.City) > 20 || len(body.City) <= 2 {
		if body.City != "" {
			logrus.Printf("City must be with in 2 to 20 letter.")
			utils.RespondError(w, http.StatusBadRequest, nil, "City must be with in 2 to 20 letter.")
			return
		}
		body.City = restaurant.City
	}

	if len(body.PinCode) != 6 {
		if body.PinCode != "" {
			logrus.Printf("PinCode must 6 digit.")
			utils.RespondError(w, http.StatusBadRequest, nil, "PinCode must 6 digit.")
			return
		}
		body.PinCode = restaurant.PinCode
	}

	if body.Lat > 90 || body.Lat < -90 {
		if body.Lat != 0 {
			logrus.Printf("Invalid Latitude.")
			utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Latitude.")
			return
		}
		body.Lat = restaurant.Lat
	}

	if body.Lng > 180 || body.Lng < -180 {
		if body.Lat != 0 {
			logrus.Printf("Invalid Longitude.")
			utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Longitude.")
			return
		}
		body.Lng = restaurant.Lng
	}

	err := dbHelper.UpdateRestaurant(restaurantId, body.Name, body.Email, body.Address, body.State, body.City, body.PinCode, body.Lat, body.Lng)
	if err != nil {
		logrus.Printf("Failed to update Restaurant: %s", err)
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to update Restaurant")
		return
	}
	logrus.Printf("Restaurant Opened successfully.")
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
		logrus.Printf("Failed to parse request body: %s", parseErr)
		utils.RespondError(w, http.StatusBadRequest, parseErr, "Failed to parse request body")
		return
	}

	exists, existsErr := dbHelper.IsRestaurantIDExists(restaurantId)
	if existsErr != nil {
		logrus.Printf("Failed to parse request body: %s", existsErr)
		utils.RespondError(w, http.StatusInternalServerError, existsErr, "Failed to check Restaurant existence")
		return
	}

	if !exists {
		logrus.Printf("Restaurant not exists.")
		utils.RespondError(w, http.StatusBadRequest, nil, "Restaurant not exists")
		return
	}

	if body.Name == "" {
		logrus.Printf("Invalid Dish Name.")
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Dish Name.")
		return
	}

	if body.Description == "" {
		logrus.Printf("Invalid Dish Description.")
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Dish Description.")
		return
	}

	if body.Quantity <= 0 {
		logrus.Printf("Invalid Dish Quantity.")
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Dish Quantity.")
		return
	}

	if body.Price <= 0 {
		logrus.Printf("Invalid Dish Price.")
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Dish Price.")
		return
	}

	if body.Discount > 100 || body.Discount < 0 {
		logrus.Printf("Invalid Discount.")
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Discount.")
		return
	}
	_, saveErr := dbHelper.CreateDish(restaurantId, adminCtx.ID, body.Name, body.Description, body.Quantity, body.Price, body.Discount)
	if saveErr != nil {
		logrus.Printf("Failed to add Restaurant Dish: %s", saveErr)
		utils.RespondError(w, http.StatusInternalServerError, saveErr, "Failed to add Restaurant Dish.")
		return
	}
	logrus.Printf("Restaurant Dishes added successfully")
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
		logrus.Printf("Failed to parse request body: %s", parseErr)
		utils.RespondError(w, http.StatusBadRequest, parseErr, "Failed to parse request body")
		return
	}

	dish, dishErr := dbHelper.GetDishByID(dishId)
	if dishErr != nil {
		logrus.Printf("Dish not exist: %s", dishErr)
		utils.RespondError(w, http.StatusBadRequest, nil, "Dish not exist")
		return
	}

	if adminCtx.CurrentRole != models.RoleAdmin && dish.CreatedBy != adminCtx.ID {
		logrus.Printf("Dish not Created by %s", adminCtx.CurrentRole)
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
		logrus.Printf("Invalid Discount.")
		utils.RespondError(w, http.StatusBadRequest, nil, "Invalid Discount.")
		return
	}

	err := dbHelper.UpdateDish(dishId, restaurantId, body.Name, body.Description, body.Quantity, body.Price, body.Discount)
	if err != nil {
		logrus.Printf("Failed update Restaurant Dish: %s", err)
		utils.RespondError(w, http.StatusInternalServerError, err, "Failed to update Restaurant Dish")
		return
	}
	logrus.Printf("Restaurant Dish Updated successfully")
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
			logrus.Printf("Failed to get Restaurant Dish: %s", err)
			utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get Restaurant Dish")
			return
		}
	} else {
		err := dbHelper.RemoveDishByUserID(dishId, restaurantId, adminCtx.ID)
		if err != nil {
			logrus.Printf("Failed to get Restaurant Dish: %s", err)
			utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get Restaurant Dish")
			return
		}
	}
	logrus.Printf("Restaurant Dish Removed successfully.")
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
			logrus.Printf("Failed to get Restaurant Dishes Count: %s", DishesCountErr)
			utils.RespondError(w, http.StatusInternalServerError, DishesCountErr, "Failed to get Restaurant Dishes Count.")
			return
		}
		Dishes, err := dbHelper.GetRestaurantDishes(restaurantId, Filters)
		if err != nil {
			logrus.Printf("Failed to get Restaurant Dishes: %s", err)
			utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get Restaurant Dishes")
			return
		}
		logrus.Printf("Get Restaurants successfully.")
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
			logrus.Printf("Failed to get Restaurant Dishes Count: %s", DishesCountErr)
			utils.RespondError(w, http.StatusInternalServerError, DishesCountErr, "Failed to get Restaurant Dishes Count.")
			return
		}
		Dishes, err := dbHelper.GetRestaurantDishesByUserID(restaurantId, adminCtx.ID, Filters)
		if err != nil {
			logrus.Printf("Failed to get Restaurant Dishes: %s", err)
			utils.RespondError(w, http.StatusInternalServerError, err, "Failed to get Restaurant Dish")
			return
		}
		logrus.Printf("Get Restaurants successfully.")
		utils.RespondJSON(w, http.StatusCreated, models.GetDishes{
			Message:    "Get Restaurants successfully.",
			Dishes:     Dishes,
			TotalCount: DishesCount,
			PageSize:   Filters.PageSize,
			PageNumber: Filters.PageNumber,
		})
	}
}
