package endpoint

import "errors"

var (
	errInternalRouteNotFound = errors.New("route not found not found")
	errApiTokenNotProvided   = errors.New("api token not provided")
	errApiTokenNotFound      = errors.New("api token not found")
	errKeyNotFound           = errors.New("key not found")
	errMethodNotAllowed      = errors.New("method not allowed")

	ErrParamNotFound    = errors.New("param not found")
	ErrInvalidParamType = errors.New("invalid param type")
	errParseError       = errors.New("parse error")
	ErrInvalidAmount    = errors.New("invalid amount")
)
