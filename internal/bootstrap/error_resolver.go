package bootstrap

import (
	authApp "github.com/soat13/ms-auth/internal/auth/application"
	authDomain "github.com/soat13/ms-auth/internal/auth/domain"
	"github.com/soat13/ms-auth/internal/shared/authz"
	userApp "github.com/soat13/ms-auth/internal/user/application"
	userDomain "github.com/soat13/ms-auth/internal/user/domain"
	errorHelper "github.com/soat13/oficina-utils/pkg/error"
	fiberHelper "github.com/soat13/oficina-utils/pkg/http/fiber"
	"github.com/soat13/oficina-utils/pkg/valueobjects/document"
	"github.com/soat13/oficina-utils/pkg/valueobjects/email"
	"github.com/soat13/oficina-utils/pkg/valueobjects/password"
	"github.com/soat13/oficina-utils/pkg/valueobjects/phone"
)

func NewErrorResolver() *errorHelper.Resolver {
	errorResolver := errorHelper.NewErrorResolver()

	errorResolver.RegisterHTTPBadRequestError(fiberHelper.ErrInvalidID)
	errorResolver.RegisterHTTPBadRequestError(document.ErrInvalidDocument)

	errorResolver.RegisterHTTPUnauthorizedError(authDomain.ErrMissingToken)
	errorResolver.RegisterHTTPUnauthorizedError(authDomain.ErrInvalidToken)
	errorResolver.RegisterHTTPUnauthorizedError(authDomain.ErrExpiredToken)
	errorResolver.RegisterHTTPUnauthorizedError(authApp.ErrInvalidCredentials)

	errorResolver.RegisterHTTPNotFoundError(userApp.ErrUserNotFound)

	errorResolver.RegisterHTTPConflictError(userApp.ErrDuplicateEmail)
	errorResolver.RegisterHTTPConflictError(userApp.ErrDuplicateDocument)

	errorResolver.RegisterHTTPUnprocessableError(userDomain.ErrInvalidUserName)
	errorResolver.RegisterHTTPUnprocessableError(authz.ErrInvalidRole)
	errorResolver.RegisterHTTPUnprocessableError(authz.ErrRolesRequired)
	errorResolver.RegisterHTTPUnprocessableError(email.ErrInvalidEmail)
	errorResolver.RegisterHTTPUnprocessableError(phone.ErrInvalidPhoneNumber)
	errorResolver.RegisterHTTPUnprocessableError(password.ErrPasswordTooShort)
	errorResolver.RegisterHTTPUnprocessableError(password.ErrPasswordTooLong)
	errorResolver.RegisterHTTPUnprocessableError(password.ErrInvalidPasswordHash)

	return errorResolver
}
