package application

import (
	"context"

	"github.com/google/uuid"
	"github.com/soat13/ms-auth/internal/shared/authz"
	"github.com/soat13/ms-auth/internal/shared/strutil"
	"github.com/soat13/ms-auth/internal/user/domain"
	"github.com/soat13/oficina-utils/pkg/valueobjects/document"
	"github.com/soat13/oficina-utils/pkg/valueobjects/email"
	"github.com/soat13/oficina-utils/pkg/valueobjects/password"
	"github.com/soat13/oficina-utils/pkg/valueobjects/phone"
)

type (
	CreateInput struct {
		Name        string
		Document    document.Document
		PhoneNumber phone.PhoneNumber
		Email       email.Email
		Password    password.Password
		Roles       authz.Roles
	}

	CreateUser struct {
		repo Repository
	}
)

func NewCreateUser(repo Repository) *CreateUser {
	return &CreateUser{repo: repo}
}

func (uc *CreateUser) Execute(ctx context.Context, in CreateInput) error {
	err := checkIfUserExists(ctx, uc.repo, in.Document, in.Email)
	if err != nil {
		return err
	}

	user, err := domain.NewUser(uuid.Nil, in.Name, in.Document, in.PhoneNumber, in.Email, in.Password, in.Roles)
	if err != nil {
		return err
	}

	return uc.repo.Create(ctx, user)
}

func checkIfUserExists(ctx context.Context, repo Repository, document document.Document, email email.Email) error {
	existsDoc, err := repo.ExistsByDocument(ctx, strutil.OnlyNumbers(document.Value))
	if err != nil {
		return err
	}
	if existsDoc {
		return ErrDuplicateDocument
	}

	existsEmail, err := repo.ExistsByEmail(ctx, email.String())
	if err != nil {
		return err
	}
	if existsEmail {
		return ErrDuplicateEmail
	}

	return nil
}
