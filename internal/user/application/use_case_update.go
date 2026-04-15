package application

import (
	"context"

	"github.com/google/uuid"
	"github.com/soat13/oficina-auth/internal/shared/authz"
	"github.com/soat13/oficina-utils/pkg/valueobjects/email"
	"github.com/soat13/oficina-utils/pkg/valueobjects/password"
	"github.com/soat13/oficina-utils/pkg/valueobjects/phone"
)

type (
	UpdateInput struct {
		ID          uuid.UUID
		Name        *string
		PhoneNumber *phone.PhoneNumber
		Email       *email.Email
		Password    *password.Password
		Roles       *authz.Roles
	}

	UpdateUser struct {
		repo Repository
	}
)

func NewUpdateUser(repo Repository) *UpdateUser {
	return &UpdateUser{repo: repo}
}

func (uc *UpdateUser) Execute(ctx context.Context, in UpdateInput) error {
	user, err := uc.repo.GetByID(ctx, in.ID)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}

	if in.Name != nil {
		if err := user.ChangeName(*in.Name); err != nil {
			return err
		}
	}
	if in.PhoneNumber != nil {
		user.ChangePhoneNumber(*in.PhoneNumber)
	}
	if in.Password != nil {
		user.ChangePassword(*in.Password)
	}
	if in.Roles != nil {
		user.ChangeRoles(*in.Roles)
	}
	if in.Email != nil && user.Email.String() != in.Email.String() {
		exists, err := uc.repo.ExistsByEmail(ctx, in.Email.String())
		if err != nil {
			return err
		}
		if exists {
			return ErrDuplicateEmail
		}
		user.ChangeEmail(*in.Email)
	}

	return uc.repo.Update(ctx, user)
}
