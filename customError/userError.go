package customErrors

import (
	"errors"
)

// Define common errors using the `errors.New` function.
var (
	ErrNotFound   = errors.New("resource not found")
	ErrForbidden  = errors.New("access forbidden")
	ErrInternal   = errors.New("internal server error")
	ErrBadRequest = errors.New("bad request")
)
