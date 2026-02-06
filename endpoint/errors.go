package endpoint

import "errors"

// Error definitions for endpoint operations.
var (
	// errInternalRouteNotFound is returned when no route matches the request.
	errInternalRouteNotFound = errors.New("route not found not found")
	// errApiTokenNotProvided is returned when the API token header is missing.
	errApiTokenNotProvided = errors.New("api token not provided")
	// errApiTokenNotFound is returned when the API token is invalid.
	errApiTokenNotFound = errors.New("api token not found")
	// errKeyNotFound is returned when a context key is not found.
	errKeyNotFound = errors.New("key not found")
	// errMethodNotAllowed is returned for unsupported HTTP methods.
	errMethodNotAllowed = errors.New("method not allowed")

	// ErrParamNotFound is returned when a required parameter is missing.
	ErrParamNotFound = errors.New("param not found")
	// ErrInvalidParamType is returned when a parameter has the wrong type.
	ErrInvalidParamType = errors.New("invalid param type")
	// errParseError is returned when JSON parsing fails.
	errParseError = errors.New("parse error")
	// ErrInvalidAmount is returned when an amount value is invalid.
	ErrInvalidAmount = errors.New("invalid amount")
)
