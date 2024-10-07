package utils

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"os"
	"regexp"
	"rms/models"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"

	"github.com/sirupsen/logrus"
	"github.com/teris-io/shortid"
	"golang.org/x/crypto/bcrypt"
)

var generator *shortid.Shortid

const generatorSeed = 1000

type FieldError struct {
	Err validator.ValidationErrors
}

func (q FieldError) GetSingleError() string {
	errorString := ""
	for _, e := range q.Err {
		errorString = "Invalid " + e.Field()
	}
	return errorString
}

type clientError struct {
	ID            string `json:"id"`
	MessageToUser string `json:"messageToUser"`
	DeveloperInfo string `json:"developerInfo"`
	Err           string `json:"error"`
	StatusCode    int    `json:"statusCode"`
	IsClientError bool   `json:"isClientError"`
}

func init() {
	n, err := rand.Int(rand.Reader, big.NewInt(generatorSeed))
	if err != nil {
		logrus.Panicf("failed to initialize utilities with random seed, %+v", err)
		return
	}

	g, err := shortid.New(1, shortid.DefaultABC, n.Uint64())

	if err != nil {
		logrus.Panicf("Failed to initialize utils package with error: %+v", err)
	}

	generator = g
}

// ParseBody parses the values from io reader to a given interface
func ParseBody(body io.Reader, out interface{}) error {
	err := json.NewDecoder(body).Decode(out)
	if err != nil {
		return err
	}

	return nil
}

// EncodeJSONBody writes the JSON body to response writer
func EncodeJSONBody(resp http.ResponseWriter, data interface{}) error {
	return json.NewEncoder(resp).Encode(data)
}

// RespondJSON sends the interface as a JSON
func RespondJSON(w http.ResponseWriter, statusCode int, body interface{}) {
	w.WriteHeader(statusCode)
	if body != nil {
		if err := EncodeJSONBody(w, body); err != nil {
			logrus.Errorf("Failed to respond JSON with error: %+v", err)
		}
	}
}

// newClientError creates structured client error response message
func newClientError(err error, statusCode int, messageToUser string, additionalInfoForDevs ...string) *clientError {
	additionalInfoJoined := strings.Join(additionalInfoForDevs, "\n")
	if additionalInfoJoined == "" {
		additionalInfoJoined = messageToUser
	}

	errorID, _ := generator.Generate()
	var errString string
	if err != nil {
		errString = err.Error()
	}
	return &clientError{
		ID:            errorID,
		MessageToUser: messageToUser,
		DeveloperInfo: additionalInfoJoined,
		Err:           errString,
		StatusCode:    statusCode,
		IsClientError: true,
	}
}

// RespondError sends an error message to the API caller and logs the error
func RespondError(w http.ResponseWriter, statusCode int, err error, messageToUser string, additionalInfoForDevs ...string) {
	logrus.Errorf("status: %d, message: %s, err: %+v ", statusCode, messageToUser, err)
	clientError := newClientError(err, statusCode, messageToUser, additionalInfoForDevs...)
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(clientError); err != nil {
		logrus.Errorf("Failed to send error to caller with error: %+v", err)
	}
}

// JwtToken generates SHA256 for a given string
func JwtToken(userId, userRoleId string) (string, error) {
	secretKey := []byte(os.Getenv("SESSION_KEY"))
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId":     userId,
		"userRoleId": userRoleId,
		"exp":        time.Now().Add(time.Hour).Unix(),
	})

	return token.SignedString(secretKey)
}

// JwtToken generates SHA256 for a given string
func ParseJwtToken(token string) error {
	secretKey := []byte(os.Getenv("SESSION_KEY"))
	claims := jwt.MapClaims{}
	// Parse the JWT token
	jwtParse, jwtErr := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return secretKey, nil
	})
	if jwtErr != nil {
		logrus.WithError(jwtErr).Errorf("failed parse JWT: %s", token)
		return jwtErr
	}
	if jwtParse != nil {
		return nil
	}
	return jwt.ErrTokenInvalidClaims
}

// IsEmailValid checks if the email provided is valid by regex.
func IsEmailValid(e string) bool {
	emailRegex := regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
	return emailRegex.MatchString(e)
}

// HashPassword returns the bcrypt hash of the password
func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashedPassword), nil
}

// CheckPassword checks if the provided password is correct or not
func CheckPassword(password, hashedPassword string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

//todo unused function val

// CheckValidation returns the current validation status
func CheckValidation(i interface{}) validator.ValidationErrors {
	v := validator.New()
	err := v.Struct(i)
	if err == nil {
		return nil
	}
	return err.(validator.ValidationErrors)
}

func CalculateDistance(lat1, lng1, lat2, lng2 float64) (float64, string) {
	// Convert degrees to radians
	var lat1R, lng1R, lat2R, lng2R = lat1 * math.Pi / 180, lng1 * math.Pi / 180, lat2 * math.Pi / 180, lng2 * math.Pi / 180
	// haversine formula for a central angle
	var a = math.Sin((lat2R-lat1R)/2)*math.Sin((lat2R-lat1R)/2) +
		math.Cos(lat1R)*math.Cos(lat2R)*
			math.Sin((lng2R-lng1R)/2)*math.Sin((lng2R-lng1R)/2)
	const RadiusOfEarth = 6371
	var distance = math.Round(200*RadiusOfEarth*math.Atan2(math.Sqrt(a), math.Sqrt(1-a))) / 100
	if distance < 1 {
		return distance * 1000, "Meter"
	}
	return distance, "Kilo Meter"
}

// TrimAll removes a given rune form given string
func TrimAll(str string, remove rune) string {
	return strings.Map(func(r rune) rune {
		if r == remove {
			return -1
		}
		return r
	}, str)
}

// TrimStringAfter trims anything after given delimiter
func TrimStringAfter(s, delim string) string {
	if idx := strings.Index(s, delim); idx != -1 {
		return s[:idx]
	}
	return s
}

// GetAddress by AddressID from Adresses
func GetUserAddressById(addressID string, addresses []models.UserAddress) (*models.UserAddress, error) {
	for _, address := range addresses {
		if address.ID == addressID {
			return &address, nil
		}
	}
	return nil, fmt.Errorf("Address not Found: %s", addressID)
}

// User with address
func ImproveUsers(rows *sql.Rows) ([]models.User, error) {
	users := make([]models.User, 0)
	for rows.Next() {
		var user models.User
		var userAddressesJSON string
		err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt, &user.CurrentRole, &userAddressesJSON)
		if err != nil {
			return nil, err
		}

		user.Password = "******"
		// Parse JSON-encoded user addresses
		var addresses []models.UserAddress
		if err := json.Unmarshal([]byte(userAddressesJSON), &addresses); err != nil {
			return nil, err
		}
		if addresses[0].ID != "" {
			user.UserAddresses = addresses
		}

		users = append(users, user)
	}
	return users, nil
}

// Filters set default value if not
func GetFilters(r *http.Request) models.Filters {
	var Filters models.Filters
	PageNumber, PageNumberErr := strconv.ParseInt(r.URL.Query().Get("pageNumber"), 10, 64)
	if PageNumberErr == nil && PageNumber != 0 {
		Filters.PageNumber = PageNumber
	} else {
		Filters.PageNumber = 0
	}
	PageSize, PageSizeErr := strconv.ParseInt(r.URL.Query().Get("pageSize"), 10, 64)
	if PageSizeErr == nil && PageSize != 0 {
		Filters.PageSize = PageSize
	} else {
		Filters.PageSize = 10
	}
	logrus.Printf("PageNumber: %d,PageSize: %d", PageNumber, PageSize)
	Name := r.URL.Query().Get("name")
	Filters.Name = Name
	Email := r.URL.Query().Get("email")
	Filters.Email = Email
	CreatedBy := r.URL.Query().Get("createdBy")
	Filters.CreatedBy = CreatedBy
	SortBy := r.URL.Query().Get("SortBy")
	//TODO remove case id From switch case because it is already added in default case
	switch SortBy {
	case "Id":
		Filters.SortBy = models.ID
	case "Name":
		Filters.SortBy = models.Name
	case "Email":
		Filters.SortBy = models.Email
	case "Created By":
		Filters.SortBy = models.CreatedBy
	default:
		Filters.SortBy = models.ID
	}
	return Filters
}

func GetDishFilters(r *http.Request) models.DishFilters {
	var Filters models.DishFilters
	PageNumber, PageNumberErr := strconv.ParseInt(r.URL.Query().Get("pageNumber"), 10, 64)
	if PageNumberErr == nil && PageNumber != 0 {
		Filters.PageNumber = PageNumber
	} else {
		Filters.PageNumber = 0
	}
	PageSize, PageSizeErr := strconv.ParseInt(r.URL.Query().Get("pageSize"), 10, 64)
	if PageSizeErr == nil && PageSize != 0 {
		Filters.PageSize = PageSize
	} else {
		Filters.PageSize = 10
	}
	MinQuantity, MinQuantityErr := strconv.ParseInt(r.URL.Query().Get("minQuantity"), 10, 64)
	if MinQuantityErr == nil && MinQuantity != 0 {
		Filters.MinQuantity = MinQuantity
	} else {
		Filters.MinQuantity = 1
	}
	MaxPrice, MaxPriceErr := strconv.ParseInt(r.URL.Query().Get("maxPrice"), 10, 64)
	if MaxPriceErr == nil && PageSize != 0 {
		Filters.MaxPrice = MaxPrice
	} else {
		Filters.MaxPrice = math.MaxInt64
	}
	MinPrice, MinPriceErr := strconv.ParseInt(r.URL.Query().Get("minPrice"), 10, 64)
	if MinPriceErr == nil && PageSize != 0 {
		Filters.MinPrice = MinPrice
	} else {
		Filters.MinPrice = 1
	}
	MaxDiscount, MaxDiscountErr := strconv.ParseInt(r.URL.Query().Get("maxDiscount"), 10, 64)
	if MaxDiscountErr == nil && PageSize != 0 {
		Filters.MaxDiscount = MaxDiscount
	} else {
		Filters.MaxDiscount = 100
	}
	MinDiscount, MinDiscountErr := strconv.ParseInt(r.URL.Query().Get("minDiscount"), 10, 64)
	if MinDiscountErr == nil && PageSize != 0 {
		Filters.MinDiscount = MinDiscount
	} else {
		Filters.MinDiscount = 1
	}
	logrus.Printf("PageNumber: %d,PageSize: %d", PageNumber, PageSize)
	Name := r.URL.Query().Get("name")
	Filters.Name = Name
	CreatedBy := r.URL.Query().Get("createdBy")
	Filters.CreatedBy = CreatedBy
	SortBy := r.URL.Query().Get("SortBy")
	switch SortBy {
	case "Id":
		Filters.SortBy = models.DishID
	case "Name":
		Filters.SortBy = models.DishName
	case "Quantity":
		Filters.SortBy = models.DishQuantity
	case "Price":
		Filters.SortBy = models.DishPrice
	case "Discount":
		Filters.SortBy = models.DishDiscount
	case "Created By":
		Filters.SortBy = models.DishCreatedBy
	default:
		Filters.SortBy = models.DishID
	}
	return Filters
}
