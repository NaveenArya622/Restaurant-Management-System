package dbHelper

import (
	"database/sql"
	"errors"
	"rms/database"
	"rms/models"
	"rms/utils"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

func CreateUser(db sqlx.Ext, name, email, password string) (string, error) {
	// language=SQL
	SQL := `INSERT INTO users(name, email, password) VALUES ($1, TRIM(LOWER($2)), $3) RETURNING id`
	var userID string
	if err := db.QueryRowx(SQL, name, email, password).Scan(&userID); err != nil {
		return "", err
	}
	return userID, nil
}

func CreateUserRole(db sqlx.Ext, userID, createdBy string, role models.Role) error {
	// language=SQL
	SQL := `INSERT INTO user_roles(user_id, created_by, role_name) VALUES ($1, $2, $3)`
	_, err := db.Exec(SQL, userID, createdBy, role)
	return err
}

func GetAddressesByUserIDs(userIDs []string) ([]models.UserAddress, error) {

	// language=SQL
	SQL := `SELECT
				ua.id,
				ua.address,
				ua.state,
				ua.city,
				ua.pin_code,
				ua.lat,
				ua.lng,
				ua.created_at,
				ua.user_id
			FROM user_address ua
			WHERE ua.archived_at IS NULL AND ua.user_id::text = any($1)`
	addresses := make([]models.UserAddress, 0)

	err := database.RMS.Select(&addresses, SQL, pq.StringArray(userIDs))

	if err != nil {
		//todo :- I think this condition is unnecessary because you will be return same error mag in both case **done**
		return nil, err
	}
	return addresses, nil
}

func CreateUserAddress(userID, address, state, city, pinCode string, lat, lng float64) error {
	arguments := []interface{}{
		userID,
		address,
		state,
		city,
		pinCode,
		lat,
		lng,
	}
	// language=SQL
	SQL := `INSERT INTO user_address(user_id, address, state, city, pin_code, lat, lng) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := database.RMS.Exec(SQL, arguments...)
	return err
}

func IsAnyRoleExist(role models.Role) (bool, error) {

	SQL := `SELECT
				count(*) > 0
       		FROM
				user_roles ur 
			WHERE
				ur.archived_at IS NULL
				AND ur.role_name = $1`
	var isUserRoleId bool
	err := database.RMS.Get(&isUserRoleId, SQL, role)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return isUserRoleId, nil
}

func GetUserRoles(userID string) ([]models.Role, error) {
	// language=SQL
	SQL := `SELECT role_name FROM user_roles WHERE user_id = $1 AND archived_at IS NULL`
	roles := make([]models.Role, 0)
	err := database.RMS.Select(&roles, SQL, userID)
	return roles, err
}

// todo address will be fetched in this single query not any other db call **done**
func GetUserBySession(sessionToken string) (*models.User, error) {
	// language=SQL
	SQL := `SELECT 
       			u.id,
       			u.name,
       			u.email,
				'********' AS password,
       			u.created_at,
				COALESCE(ua.id,gen_random_uuid()) AS address_id,
				COALESCE(ua.address,'') AS address,
				COALESCE(ua.state,'') AS state,
				COALESCE(ua.city,'') AS city,
				COALESCE(ua.pin_code,'') AS pin_code,
				COALESCE(ua.lat,0) AS lat,
				COALESCE(ua.lng,0) AS lng,
				COALESCE(ua.created_at,now()) AS address_created_at
			FROM users u
			JOIN user_session us on u.id = us.user_id
			JOIN user_roles ur on u.id = ur.user_id
			LEFT JOIN user_address ua on u.id = ua.user_id
			WHERE u.archived_at IS NULL AND ur.archived_at IS NULL AND us.session_token = $1
			GROUP BY ua.id,u.id`
	var users []models.UserWithAddress
	err := database.RMS.Select(&users, SQL, sessionToken)

	if err != nil {
		//todo :- I think this condition is unnecessary because you will be return same error mag in both case **done**
		return nil, err
	}
	user := utils.GetUser(users)
	user.Roles, err = GetUserRoles(user.ID)
	if err != nil {
		//todo :- I think this condition is unnecessary because you will be return same error mag in both case **done**
		return nil, err
	}
	return &user, nil
}

func UserHaveMultipleRoles(id string) (bool, error) {
	SQL := `SELECT
				count(*) > 1
       		FROM
				user_roles ur
			WHERE
				ur.archived_at IS NULL
				AND ur.user_id = $1`
	var multipleRoles bool
	err := database.RMS.Get(&multipleRoles, SQL, id)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return multipleRoles, nil
}

func CreateUserSession(db sqlx.Ext, userID, sessionToken string) error {
	// language=SQL
	SQL := `INSERT INTO user_session(user_id, session_token) VALUES ($1, $2)`
	_, err := db.Exec(SQL, userID, sessionToken)
	return err
}

func GetUserIDByPassword(email, password string) (string, error) {
	// language=SQL
	SQL := `SELECT
				u.id,
       			u.password
       		FROM
				users u JOIN user_roles ur on u.id = ur.user_id
			WHERE
				u.archived_at IS NULL
				AND ur.archived_at IS NULL
				AND u.email = TRIM(LOWER($1))`
	var user models.User
	err := database.RMS.Get(&user, SQL, email)
	if err != nil {
		//TODO:- remove if condition **DONE**
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	// compare password
	if passwordErr := utils.CheckPassword(password, user.Password); passwordErr != nil {
		return "", passwordErr
	}
	return user.ID, nil
}

func DeleteSessionToken(token string) error {
	// language=SQL
	SQL := `DELETE FROM user_session WHERE session_token = $1`
	_, err := database.RMS.Exec(SQL, token)
	return err
}

func IsUserRoleExists(email string, role models.Role) (bool, error) {
	// language=SQL
	//todo alternate use count(*) > 0 **DONE**
	SQL := `SELECT
				count(*) > 0
       		FROM
				users u JOIN user_roles ur on u.id = ur.user_id
			WHERE
				u.archived_at IS NULL
				AND ur.archived_at IS NULL
				AND u.email = TRIM(LOWER($1))
				AND ur.role_name = $2`
	var isUserRoleId bool
	err := database.RMS.Get(&isUserRoleId, SQL, email, role)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return isUserRoleId, nil
}

func IsUserRoleWithUserIDExists(id string, role models.Role) (bool, error) {
	// language=SQL
	SQL := `SELECT
				id
       		FROM
				user_roles
			WHERE
				archived_at IS NULL
				AND user_id = $1
				AND role_name = $2`
	var userRoleId string
	err := database.RMS.Get(&userRoleId, SQL, id, role)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func IsUserExists(email string) (string, error) {
	// language=SQL
	SQL := `SELECT id FROM users WHERE email = TRIM(LOWER($1)) AND archived_at IS NULL`
	var userId string
	err := database.RMS.Get(&userId, SQL, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return userId, nil
}

func UpdateUserInfo(userID, newName, newEmail, newPassword string) error {
	arguments := []interface{}{
		newName,
		newEmail,
		newPassword,
		userID,
	}
	// language=SQL
	SQL := `UPDATE users 
		SET name = $1, 
			email = TRIM(LOWER($2)),
			password = $3
		WHERE id = $4`
	_, err := database.RMS.Exec(SQL, arguments...)
	return err
}

func RemoveRoleByAdminID(db sqlx.Ext, userID, createdBy string, role models.Role) error {
	// language=SQL
	SQL := `UPDATE user_roles 
		SET archived_at = $1
		WHERE user_id = $2 AND created_by = $3 AND role_name = $4
		RETURNING id;`
	_, err := db.Exec(SQL, time.Now(), userID, createdBy, role)
	return err
}

func RemoveUser(db sqlx.Ext, userID string) error {
	// language=SQL
	SQL := `UPDATE users 
		SET archived_at = $1
		WHERE user_id = $2`
	_, err := db.Exec(SQL, time.Now(), userID)
	return err
}

func RemoveRole(db sqlx.Ext, userID string, role models.Role) error {
	// language=SQL
	SQL := `UPDATE user_roles 
		SET archived_at = $1
		WHERE user_id = $2 AND role_name = $3`
	_, err := db.Exec(SQL, time.Now(), userID, role)
	return err
}

func UpdateUserAddress(AddressID, Address, State, City, PinCode string, Lat, Lng float64) error {
	arguments := []interface{}{
		Address,
		State,
		City,
		PinCode,
		Lat,
		Lng,
		AddressID,
	}
	// language=SQL
	SQL := `UPDATE user_address
		SET address = $1, 
			state = $2,
			city = $3, 
			pin_code = $4,
			lat = $5, 
			lng = $6
		WHERE id = $7`
	_, err := database.RMS.Exec(SQL, arguments...)
	return err
}

func GetUsersByAdminID(createdBy string, role models.Role, Filters models.Filters) ([]models.User, error) {
	arguments := []interface{}{
		createdBy,
		role,
		Filters.Name,
		Filters.Email,
		Filters.SortBy,
		Filters.PageSize,
		Filters.PageSize * Filters.PageNumber,
	}
	// language=SQL
	SQL := `SELECT 
       			u.id,
       			u.name,
       			u.email,
       			u.created_at,
				ucr.role_name AS user_current_role
			FROM users u
			JOIN user_roles ucr on u.id = ucr.user_id
			WHERE u.archived_at IS NULL AND ucr.archived_at IS NULL AND ucr.created_by=$1 AND ucr.role_name=$2 AND
			 	u.name ILIKE '%' || $3 || '%' AND  u.email ILIKE '%' || $4 || '%'
			ORDER BY $5
			LIMIT $6
			OFFSET $7`

	users := make([]models.User, 0)
	err := database.RMS.Select(&users, SQL, arguments...)

	if err != nil {
		return nil, err
	}
	userIDs := utils.GetValuesFromUser(users, "ID")

	addresses, addressErr := GetAddressesByUserIDs(userIDs)

	if addressErr != nil {
		return nil, err
	}
	return utils.UpdateUserAddress(users, addresses)
}

func GetUserCountByAdminID(createdBy string, role models.Role, Filters models.Filters) (int64, error) {
	arguments := []interface{}{
		createdBy,
		role,
		Filters.Name,
		Filters.Email,
	}
	SQL := `SELECT 
       			COUNT(id)
			FROM users u
			JOIN user_roles ucr on u.id = ucr.user_id
			WHERE u.archived_at IS NULL AND ucr.archived_at IS NULL AND ucr.created_by=$1 AND ucr.role_name=$2 AND
			u.name ILIKE '%' || $3 || '%' AND  u.email ILIKE '%' || $4 || '%'`
	var count int64
	err := database.RMS.Get(&count, SQL, arguments...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return count, nil
}

func GetUserCount(role models.Role, Filters models.Filters) (int64, error) {
	arguments := []interface{}{
		role,
		Filters.CreatedBy,
		Filters.Name,
		Filters.Email,
	}
	SQL := `SELECT 
       			COUNT(ucr.id)
			FROM users u
			JOIN user_roles ucr on u.id = ucr.user_id
			WHERE u.archived_at IS NULL AND ucr.archived_at IS NULL AND ucr.role_name=$1 AND ucr.created_by::text ILIKE '%' || $2 || '%' AND
			u.name ILIKE '%' || $3 || '%' AND  u.email ILIKE '%' || $4 || '%'`
	var count int64
	err := database.RMS.Get(&count, SQL, arguments...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return count, nil
}

func GetUsers(role models.Role, Filters models.Filters) ([]models.User, error) {
	arguments := []interface{}{
		role,
		Filters.CreatedBy,
		Filters.Name,
		Filters.Email,
		Filters.SortBy,
		Filters.PageSize,
		Filters.PageSize * Filters.PageNumber,
	}
	// language=SQL
	//todo avoid using json agg as much as possible because it is heavy **no need**
	SQL := `SELECT 
       			u.id,
       			u.name,
       			u.email,
       			u.created_at,
				ucr.role_name AS user_current_role
			FROM users u
			JOIN user_roles ucr on u.id = ucr.user_id
			WHERE u.archived_at IS NULL AND ucr.archived_at IS NULL AND ucr.role_name=$1 AND ucr.created_by::text ILIKE '%' || $2 || '%' AND
			u.name ILIKE '%' || $3 || '%' AND  u.email ILIKE '%' || $4 || '%'
			ORDER BY $5
			LIMIT $6
			OFFSET $7`

	users := make([]models.User, 0)
	err := database.RMS.Select(&users, SQL, arguments...)

	if err != nil {
		return nil, err
	}
	userIDs := utils.GetValuesFromUser(users, "ID")

	addresses, addressErr := GetAddressesByUserIDs(userIDs)
	if addressErr != nil {
		return nil, err
	}

	return utils.UpdateUserAddress(users, addresses)

}
