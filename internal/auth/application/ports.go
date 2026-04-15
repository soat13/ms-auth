package application

import (
	"context"

	"github.com/google/uuid"
	authDomain "github.com/soat13/oficina-auth/internal/auth/domain"
	"github.com/soat13/oficina-auth/internal/shared/authz"
	"github.com/soat13/oficina-utils/pkg/valueobjects/password"
)

type (
	UserView struct {
		ID       uuid.UUID
		Name     string
		Email    string
		Password password.Password
		Roles    authz.Roles
	}

	UserReader interface {
		GetByDocument(ctx context.Context, document string) (*UserView, error)
	}

	TokenService interface {
		Generate(ctx context.Context, user *UserView) (authDomain.Token, error)
		Validate(ctx context.Context, token string) (authDomain.Claims, error)
	}
)
