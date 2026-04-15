package application

import (
	"context"

	"github.com/google/uuid"
	"github.com/soat13/oficina-auth/internal/shared/authz"
	"github.com/soat13/oficina-auth/internal/user/domain"
	"github.com/soat13/oficina-utils/pkg/pagination"
	"github.com/soat13/oficina-utils/pkg/valueobjects/document"
	"github.com/soat13/oficina-utils/pkg/valueobjects/email"
	"github.com/soat13/oficina-utils/pkg/valueobjects/phone"
)

type (
	UserView struct {
		ID          uuid.UUID
		Name        string
		Document    document.Document
		Email       email.Email
		PhoneNumber phone.PhoneNumber
		Roles       authz.Roles
	}

	Repository interface {
		Create(ctx context.Context, u *domain.User) error
		Update(ctx context.Context, u *domain.User) error
		Delete(ctx context.Context, id uuid.UUID) error
		GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
		GetByEmail(ctx context.Context, email string) (*domain.User, error)
		List(ctx context.Context, pager pagination.Pagination) ([]*domain.User, error)
		ExistsByEmail(ctx context.Context, email string) (bool, error)
		ExistsByDocument(ctx context.Context, document string) (bool, error)
	}
)

func toView(u *domain.User) UserView {
	return UserView{
		ID:          u.ID,
		Name:        u.Name,
		Document:    u.Document,
		Email:       u.Email,
		PhoneNumber: u.PhoneNumber,
		Roles:       u.Roles,
	}
}
