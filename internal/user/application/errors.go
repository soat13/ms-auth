package application

import "errors"

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrDuplicateEmail    = errors.New("user with this email already exists")
	ErrDuplicateDocument = errors.New("user with this document already exists")
)
