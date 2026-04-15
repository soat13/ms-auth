package domain

import "errors"

var (
	ErrMissingToken = errors.New("missing authorization token")
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)
