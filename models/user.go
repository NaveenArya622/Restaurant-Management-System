package models

import (
	"time"
)

type Role string

const (
	RoleAdmin    Role = "admin"
	RoleSubAdmin Role = "sub-admin"
	RoleUser     Role = "user"
)

func (r Role) IsValid() bool {
	return r == RoleAdmin || r == RoleSubAdmin || r == RoleUser
}

func (r Role) Contains(roles []Role) bool {
	for _, role := range roles {
		return role == r && r.IsValid()
	}
	return false
}

type SortedBy string

const (
	ID        SortedBy = "id"
	Name      SortedBy = "name"
	Email     SortedBy = "email"
	CreatedBy SortedBy = "created_by"
)

func (s SortedBy) IsValid() bool {
	return s == ID || s == Name || s == Email || s == CreatedBy
}

// User

type User struct {
	ID            string        `json:"id" db:"id"`
	Name          string        `json:"name" db:"name"`
	Email         string        `json:"email" db:"email"`
	Password      string        `json:"password" db:"password"`
	CreatedAt     time.Time     `json:"createdAt" db:"created_at"`
	CurrentRole   Role          `json:"-" db:"user_current_role"`
	Roles         []Role        `json:"-" db:"roles"`
	UserAddresses []UserAddress `json:"Addresses" db:"user_addresses"`
}

type UserWithAddress struct {
	ID               string    `json:"id" db:"id"`
	Name             string    `json:"name" db:"name"`
	Email            string    `json:"email" db:"email"`
	Password         string    `json:"password" db:"password"`
	CreatedAt        time.Time `json:"createdAt" db:"created_at"`
	Roles            []Role    `json:"-" db:"roles"`
	CurrentRole      Role      `json:"-" db:"user_current_role"`
	AddressID        string    `json:"addressId" db:"address_id"`
	Address          string    `json:"address" db:"address"`
	State            string    `json:"state" db:"state"`
	City             string    `json:"city" db:"city"`
	PinCode          string    `json:"pinCode" db:"pin_code"`
	Lat              float64   `json:"lat" db:"lat"`
	Lng              float64   `json:"lng" db:"lng"`
	AddressCreatedAt time.Time `json:"addressCreatedAt" db:"address_created_at"`
}

type Filters struct {
	PageNumber int64
	PageSize   int64
	Name       string
	Email      string
	CreatedBy  string
	SortBy     SortedBy
}

type RegisterUserBody struct {
	Name     string `json:"name" db:"name"`
	Email    string `json:"email" db:"email"`
	Password string `json:"password" db:"password"`
}

type GetUsers struct {
	Message    string `json:"message"`
	Users      []User `json:"users"`
	TotalCount int64  `json:"totalCount"`
	PageNumber int64  `json:"pageNumber"`
	PageSize   int64  `json:"pageSize"`
}

type GetUser struct {
	Message string `json:"message"`
	User    User   `json:"info"`
}

type Login struct {
	Token   string `json:"token"`
	Type    string `json:"tokenType"`
	Message string `json:"message"`
}

type LoginBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type GetSubAdmins struct {
	Message    string `json:"message"`
	SubAdmins  []User `json:"subAdmins"`
	TotalCount int64  `json:"totalCount"`
	PageNumber int64  `json:"pageNumber"`
	PageSize   int64  `json:"pageSize"`
}

// Address

type UserAddress struct {
	ID        string    `json:"id" db:"id"`
	Address   string    `json:"address" db:"address"`
	State     string    `json:"state" db:"state"`
	City      string    `json:"city" db:"city"`
	PinCode   string    `json:"pinCode" db:"pin_code"`
	Lat       float64   `json:"lat" db:"lat"`
	Lng       float64   `json:"lng" db:"lng"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UserId    string    `json:"-" db:"user_id"`
}

type AddUserAddressBody struct {
	Address string  `json:"address" db:"address"`
	State   string  `json:"state" db:"state"`
	City    string  `json:"city" db:"city"`
	PinCode string  `json:"pinCode" db:"pin_code"`
	Lat     float64 `json:"lat" db:"lat"`
	Lng     float64 `json:"lng" db:"lng"`
}

// Message

type Message struct {
	Message string `json:"message"`
}
