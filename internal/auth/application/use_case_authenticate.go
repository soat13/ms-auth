package application

import (
	"context"

	authDomain "github.com/soat13/ms-auth/internal/auth/domain"
	"github.com/soat13/ms-auth/internal/shared/strutil"
	"github.com/soat13/oficina-utils/pkg/valueobjects/document"
)

type (
	AuthenticateInput struct {
		Document document.Document
		Password string
	}

	AuthenticateOutput struct {
		Token authDomain.Token
		User  UserView
	}

	AuthenticateUser struct {
		userReader   UserReader
		tokenService TokenService
	}
)

func NewAuthenticateUser(userReader UserReader, tokenService TokenService) *AuthenticateUser {
	return &AuthenticateUser{
		userReader:   userReader,
		tokenService: tokenService,
	}
}

func (uc *AuthenticateUser) Execute(ctx context.Context, in AuthenticateInput) (AuthenticateOutput, error) {
	docStr := strutil.OnlyNumbers(in.Document.Value)
	user, err := uc.userReader.GetByDocument(ctx, docStr)
	if err != nil {
		return AuthenticateOutput{}, err
	}

	if user == nil || !user.Password.Matches(in.Password) {
		return AuthenticateOutput{}, ErrInvalidCredentials
	}

	token, err := uc.tokenService.Generate(ctx, user)
	if err != nil {
		return AuthenticateOutput{}, err
	}

	return AuthenticateOutput{Token: token, User: *user}, nil
}
